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
	Post_score_average  float64
	Best_sentiment_meme string
}

func (self *Post) encode() []byte {
	message_id := encode8(PostOP)
	message := encode_string(self.Post)

	return append(message_id, message...)
}

func (self *Post) fromEncoding(code []byte) error {
	_, start := decode8(code)

	post, _ := decode_string(code[start:])

	self.Post = post

	return nil
}

func (self *Comment) encode() []byte {
	message_id := encode8(CommentOP)
	message := encode_string(self.Comment)

	return append(message_id, message...)
}

func (self *Comment) fromEncoding(code []byte) error {
	_, start := decode8(code)

	comment, _ := decode_string(code[start:])

	self.Comment = comment

	return nil
}

func (self *Error) encode() []byte {
	message_id := encode8(ErrorOP)
	message := encode_string(self.Message)
	return append(message_id, message...)
}

func (self *Error) fromEncoding(code []byte) error {

	_, start := decode8(code)

	message, _ := decode_string(code[start:])

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

	message_id := encode8(ResponseOP)
	score := encodeF64(self.Post_score_average)
	meme := encode_string(self.Best_sentiment_meme) //TODO Modify to add new fields

	return append_slices([][]byte{message_id, score, meme})

}

func (self *Response) fromEncoding(code []byte) error {
	_, start := decode8(code)
	avg, n := decodeF64(code[start:])
	start += n
	meme, _ := decode_string(code[start:])

	self.Post_score_average = avg
	self.Best_sentiment_meme = meme

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
