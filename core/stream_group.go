package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/yomorun/yomo/core/frame"
	"github.com/yomorun/yomo/core/metadata"
	"github.com/yomorun/yomo/core/yerr"
	"golang.org/x/exp/slog"
)

// StreamGroup is the group of stream includes ControlStream amd DataStream.
// One Connection has many DataStream and only one ControlStream, ControlStream authenticates
// Connection and recevies HandshakeFrame and CloseStreamFrame to create DataStream or close
// stream. the ControlStream always the first stream established between server and client.
type StreamGroup struct {
	conn          quic.Connection
	group         sync.WaitGroup
	controlStream frame.ReadWriter
	logger        *slog.Logger
}

// NewStreamGroup returns StreamGroup.
func NewStreamGroup(conn quic.Connection, controlStream frame.ReadWriter, logger *slog.Logger) *StreamGroup {
	group := &StreamGroup{
		conn:          conn,
		controlStream: controlStream,
		logger:        logger,
	}
	return group
}

// VerifyAuthentication verify Authentication from client in verifyFunc.
func (g *StreamGroup) VerifyAuthentication(verifyFunc func(*frame.AuthenticationFrame) (bool, error)) error {
	first, err := g.controlStream.ReadFrame()
	if err != nil {
		return err
	}
	f, ok := first.(*frame.AuthenticationFrame)
	if !ok {
		return err
	}
	ok, err = verifyFunc(f)
	if err != nil {
		return err
	}
	if !ok {
		errAuth := fmt.Errorf("yomo: authentication failed, client credential name is %s", f.AuthName())
		return g.authFailed(errAuth)
	}
	return g.authOK()
}

func (g *StreamGroup) authFailed(se error) error {
	resp := frame.NewAuthenticationRespFrame(false, se.Error())

	err := g.controlStream.WriteFrame(resp)
	if err != nil {
		return err
	}

	err = g.conn.CloseWithError(quic.ApplicationErrorCode(yerr.ErrorCodeRejected), se.Error())

	return err
}

func (g *StreamGroup) authOK() error {
	return g.controlStream.WriteFrame(frame.NewAuthenticationRespFrame(true, ""))
}

// Run run contextFunc with connector.
// Run continus read HandshakeFrame and CloseStreamFrame from controlStream to create DataStream
// or close DataStream. Adding new dataStream to connector and handle it in contextFunc if create one,
// Removing from connector and close it if closing a dataStream.
func (g *StreamGroup) Run(connector *Connector, mb metadata.Builder, contextFunc func(c *Context)) error {
	for {
		f, err := g.controlStream.ReadFrame()
		if err != nil {
			return err
		}

		switch ff := f.(type) {
		// client requests a new stream.
		case *frame.HandshakeFrame:
			stream, err := g.conn.OpenStreamSync(context.Background())
			if err != nil {
				return err
			}
			stream.Write(frame.NewHandshakeAckFrame().Encode())

			md, err := mb.Build(ff)
			if err != nil {
				g.logger.Warn("Build Metadata Failed", "error", err)
				continue
			}

			dataStream := newDataStream(
				ff.Name(),
				ff.ID(),
				StreamType(ff.StreamType()),
				md,
				stream,
				ff.ObserveDataTags(),
				g.logger,
				g.controlStream,
			)
			connector.Add(dataStream.ID(), dataStream)
			g.group.Add(1)

			go func() {
				defer g.group.Done()

				c := newContext(dataStream, g.logger)
				defer c.Clean()

				contextFunc(c)
			}()
		// client requests to close an exist stream.
		case *frame.CloseStreamFrame:
			stream, ok, err := connector.Get(ff.StreamID())
			if err != nil {
				g.logger.Error("Connector Get Error", err, "stream_id", ff.StreamID())
			}
			if !ok {
				continue
			}
			if err := stream.Close(); err != nil {
				g.logger.Error(
					"Close Stream Error",
					err,
					"stream_name", stream.Name(),
					"stream_type", stream.StreamType().String(),
					"stream_id", stream.ID(),
					"close_reason", ff.Reason(),
				)
			}
			g.logger.Debug(
				"Client Close Stream",
				"stream_name", stream.Name(),
				"stream_type", stream.StreamType().String(),
				"stream_id", stream.ID(),
				"close_reason", ff.Reason(),
			)
			connector.Remove(ff.StreamID())
		}
	}
}

// Wait waits all dataStream down.
func (g *StreamGroup) Wait() { g.group.Wait() }