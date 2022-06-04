package protocol

import "fmt"

type Encodable interface {
	encode() []byte
	fromEncoding([]byte) error
	// implementsEncodable() can be added as a method to force explicit interface implementation. It doesn't do anything
}

const (
	PostOP            uint8 = 0
	CommentOP               = 1
	PostFinishedOP          = 2
	CommentFinishedOP       = 3
	ErrorOP                 = 4
	ResponseOP              = 5
)

type Post struct { //Implements Encodable
	Post string
}

type Comment struct { //Implements Encodable
	Comment string
}

type Error struct { //Implements Encodable
	Message string
}

type PostFinished struct { //Implements Encodable
}

type CommentFinished struct { //Implements Encodable
}

type Response struct { //Implements Encodable
	PostScoreAvg      float64
	BestSentimentMeme []byte
	SchoolMemes       []string
}

func (self *Post) encode() []byte {
	messageId := encode8(PostOP)
	message := encodeString(self.Post)

	return append(messageId, message...)
}

func (self *Post) fromEncoding(code []byte) error {
	_, start := decode8(code)

	post, _ := decodeString(code[start:])

	self.Post = post

	return nil
}

func (self *Comment) encode() []byte {
	messageId := encode8(CommentOP)
	message := encodeString(self.Comment)

	return append(messageId, message...)
}

func (self *Comment) fromEncoding(code []byte) error {
	_, start := decode8(code)

	comment, _ := decodeString(code[start:])

	self.Comment = comment

	return nil
}

func (self *Error) encode() []byte {
	messageId := encode8(ErrorOP)
	message := encodeString(self.Message)
	return append(messageId, message...)
}

func (self *Error) fromEncoding(code []byte) error {

	_, start := decode8(code)

	message, _ := decodeString(code[start:])

	self.Message = message

	return nil
}

func (self *PostFinished) encode() []byte {
	return encode8(PostFinishedOP)
}

func (self *PostFinished) fromEncoding(code []byte) error {
	return nil
}

func (self *CommentFinished) encode() []byte {
	return encode8(CommentFinishedOP)
}

func (self *CommentFinished) fromEncoding(code []byte) error {
	return nil
}

func (self *Response) encode() []byte {

	messageId := encode8(ResponseOP)
	score := encodeF64(self.PostScoreAvg)
	meme := encodeByteSlice(self.BestSentimentMeme)
	schoolMemes := encodeStringSlice(self.SchoolMemes)

	return appendSlices([][]byte{messageId, score, meme, schoolMemes})

}

func (self *Response) fromEncoding(code []byte) error {
	_, start := decode8(code)
	avg, n := decodeF64(code[start:])
	start += n
	meme, n := decodeByteSlice(code[start:])
	start += n
	schoolMemes, _ := decodeStringSlice(code[start:])

	self.PostScoreAvg = avg
	self.BestSentimentMeme = meme
	self.SchoolMemes = schoolMemes

	return nil
}

func Encode(message Encodable) []byte {
	return message.encode()
}

func Decode(code []byte) (Encodable, error) {
	var err error
	var msg Encodable

	id, _ := decode8(code)
	message_id := uint8(id)

	switch message_id {

	case PostOP:
		msg = &Post{}
	case CommentOP:
		msg = &Comment{}
	case ErrorOP:
		msg = &Error{}
	case PostFinishedOP:
		msg = &PostFinished{}
	case CommentFinishedOP:
		msg = &CommentFinished{}
	case ResponseOP:
		msg = &Response{}
	default:
		return nil, fmt.Errorf("Unknown received message\n")
	}

	err = msg.fromEncoding(code) //Including message_id
	if err != nil {
		return nil, err
	}

	return msg, nil
}
