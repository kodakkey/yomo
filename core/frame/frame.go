// Package frame defines frames for yomo.
package frame

import (
	"fmt"
	"io"
)

// Frame is the minimum unit required for Yomo to run.
// Yomo transmits various instructions and data through the frames.
//
// following frame types are supported by Yomo:
//  1. AuthenticationFrame
//  2. AuthenticationAckFrame
//  3. DataFrame
//  4. HandshakeFrame
//  5. HandshakeRejectedFrame
//  6. HandshakeAckFrame
//  7. RejectedFrame
//  8. BackflowFrame
//
// Read frame comments to understand the role of the frame.
type Frame interface {
	// Type returns the type of frame.
	Type() Type
}

// Type defined The type of frame.
type Type byte

// AuthenticationFrame is used to authenticate the client,
// once the connection is established, the client immediately, sends information to the server,
// server gets the way to authenticate according to AuthName and use AuthPayload to do a authentication.
// AuthenticationFrame is transmit on ControlStream.
//
// Reading the `auth.Authentication` interface will help you understand how AuthName and AuthPayload work.
type AuthenticationFrame struct {
	// AuthName.
	AuthName string
	// AuthPayload.
	AuthPayload string
}

// Type returns the type of AuthenticationFrame.
func (f *AuthenticationFrame) Type() Type { return TypeAuthenticationFrame }

// AuthenticationAckFrame is used to confirm that the client is authorized to access the requested DataStream from
// ControlStream, AuthenticationAckFrame is transmit on ControlStream.
// If the client-side receives this frame, it indicates that authentication was successful.
type AuthenticationAckFrame struct{}

// Type returns the type of AuthenticationAckFrame.
func (f *AuthenticationAckFrame) Type() Type { return TypeAuthenticationAckFrame }

// DataFrame carries tagged data to transmit across DataStream.
type DataFrame struct {
	// Metadata stores additional data beyond the Payload,
	// it is an map[string]string{} that be encoded in msgpack.
	Metadata []byte
	// Tag is used for data router.
	Tag Tag
	// Payload is the data to transmit.
	Payload []byte
}

// Type returns the type of DataFrame.
func (f *DataFrame) Type() Type { return TypeDataFrame }

// The HandshakeFrame is the frame through which the client obtains a new data stream from the server.
// It include essential details required for the creation of a fresh DataStream.
// The server then generates the DataStream utilizing this provided information.
type HandshakeFrame struct {
	// Name is the name of the dataStream that will be created.
	Name string
	// ID is the ID of the dataStream that will be created.
	ID string
	// StreamType is the StreamType of the dataStream that will be created.
	StreamType byte
	// ObserveDataTags is the ObserveDataTags of the dataStream that will be created.
	ObserveDataTags []Tag
	// Metadata is the Metadata of the dataStream that will be created.
	Metadata []byte
}

// Type returns the type of HandshakeFrame.
func (f *HandshakeFrame) Type() Type { return TypeHandshakeFrame }

// HandshakeAckFrame is used to ack handshake, If handshake successful, The server will
// send HandshakeAckFrame to the new DataStream, That means the initial frame received by the new DataStream must be the HandshakeAckFrame.
type HandshakeAckFrame struct {
	StreamID string
}

// Type returns the type of HandshakeAckFrame.
func (f *HandshakeAckFrame) Type() Type { return TypeHandshakeAckFrame }

// HandshakeRejectedFrame is employed to reject a handshake. It is transmitted over the ControlStream
type HandshakeRejectedFrame struct {
	// ID is the ID of DataStream be rejected.
	ID string
	// Message contains the reason why the handshake was not successful.
	Message string
}

// Type returns the type of HandshakeRejectedFrame.
func (f *HandshakeRejectedFrame) Type() Type { return TypeHandshakeRejectedFrame }

// The BackflowFrame is used to receive the processed result of a DataStream with StreamFunction type
// and forward it to a DataStream with StreamSource type.
type BackflowFrame struct {
	// Tag is used for data router.
	Tag Tag
	// Carriage is the data to transmit.
	Carriage []byte
}

// Type returns the type of BackflowFrame.
func (f *BackflowFrame) Type() Type { return TypeBackflowFrame }

// RejectedFrame is used to reject a ControlStream request.
type RejectedFrame struct {
	// Message encapsulates the rationale behind the rejection of the request.
	Message string
}

// Type returns the type of RejectedFrame.
func (f *RejectedFrame) Type() Type { return TypeRejectedFrame }

// GoawayFrame is is used by server to evict a connection.
type GoawayFrame struct {
	// Message contains the reason why the connection be evicted.
	Message string
}

// Type returns the type of GoawayFrame.
func (f *GoawayFrame) Type() Type { return TypeGoawayFrame }

const (
	TypeAuthenticationFrame    Type = 0x03 // TypeAuthenticationFrame is the type of AuthenticationFrame.
	TypeAuthenticationAckFrame Type = 0x11 // TypeAuthenticationAckFrame is the type of AuthenticationAckFrame.
	TypeDataFrame              Type = 0x3F // TypeDataFrame is the type of DataFrame.
	TypeHandshakeFrame         Type = 0x31 // TypeHandshakeFrame is the type of PayloadFrame.
	TypeHandshakeRejectedFrame Type = 0x14 // TypeHandshakeRejectedFrame is the type of HandshakeRejectedFrame.
	TypeHandshakeAckFrame      Type = 0x29 // TypeHandshakeAckFrame is the type of HandshakeAckFrame.
	TypeRejectedFrame          Type = 0x39 // TypeRejectedFrame is the type of RejectedFrame.
	TypeBackflowFrame          Type = 0x2D // TypeBackflowFrame is the type of BackflowFrame.
	TypeGoawayFrame            Type = 0x2E // TypeGoawayFrame is the type of GoawayFrame.
)

var frameTypeStringMap = map[Type]string{
	TypeAuthenticationFrame:    "AuthenticationFrame",
	TypeAuthenticationAckFrame: "AuthenticationAckFrame",
	TypeDataFrame:              "DataFrame",
	TypeHandshakeFrame:         "HandshakeFrame",
	TypeHandshakeRejectedFrame: "HandshakeRejectedFrame",
	TypeHandshakeAckFrame:      "HandshakeAckFrame",
	TypeRejectedFrame:          "RejectedFrame",
	TypeBackflowFrame:          "BackflowFrame",
	TypeGoawayFrame:            "GoawayFrame",
}

// String returns a human-readable string which represents the frame type.
// The string can be used for debugging or logging purposes.
func (f Type) String() string {
	frameString, ok := frameTypeStringMap[f]
	if ok {
		return frameString
	}
	return "UnknownFrame"
}

var frameTypeNewFuncMap = map[Type]func() Frame{
	TypeAuthenticationFrame:    func() Frame { return new(AuthenticationFrame) },
	TypeAuthenticationAckFrame: func() Frame { return new(AuthenticationAckFrame) },
	TypeDataFrame:              func() Frame { return new(DataFrame) },
	TypeHandshakeFrame:         func() Frame { return new(HandshakeFrame) },
	TypeHandshakeRejectedFrame: func() Frame { return new(HandshakeRejectedFrame) },
	TypeHandshakeAckFrame:      func() Frame { return new(HandshakeAckFrame) },
	TypeRejectedFrame:          func() Frame { return new(RejectedFrame) },
	TypeBackflowFrame:          func() Frame { return new(BackflowFrame) },
	TypeGoawayFrame:            func() Frame { return new(GoawayFrame) },
}

// NewFrame creates a new frame from Type.
func NewFrame(f Type) (Frame, error) {
	newFunc, ok := frameTypeNewFuncMap[f]
	if ok {
		return newFunc(), nil
	}
	return nil, fmt.Errorf("frame: cannot new a frame from %c", f)
}

// PacketReadWriter reads packets from the io.Reader and writes packets to the io.Writer.
// If read failed, return the frameType, the data of the packet and an error.
type PacketReadWriter interface {
	ReadPacket(io.Reader) (Type, []byte, error)
	WritePacket(io.Writer, Type, []byte) error
}

// Codec encodes and decodes byte array to frame.
type Codec interface {
	// Decode decodes byte array to frame.
	Decode([]byte, Frame) error
	// Encode encodes frame to byte array.
	Encode(Frame) ([]byte, error)
}

// Tag tags data and can be used for data routing.
type Tag = uint32

// ReadWriteCloser is the interface that groups the ReadFrame, WriteFrame and Close methods.
type ReadWriteCloser interface {
	Reader
	Writer
	Close() error
}

// ReadWriter is the interface that groups the ReadFrame and WriteFrame methods.
type ReadWriter interface {
	Reader
	Writer
}

// Writer is the interface that wraps the WriteFrame method, it writes
// frame to the underlying data stream.
type Writer interface {
	// WriteFrame writes frame to underlying stream.
	WriteFrame(Frame) error
}

// Reader reads frame from underlying stream.
type Reader interface {
	// ReadFrame reads a frame, if an error occurs, the returned error will not be empty,
	// and the returned frame will be nil.
	ReadFrame() (Frame, error)
}
