package store

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/dustinevan/mongo/bsoncv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type MongoCollection interface {
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (Cursor, error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) (Decoder, error)
	FindOneAndDecode(ctx context.Context, filter interface{}, destination interface{}) (bool, error)
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (Cursor, error)
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (string, error)
}

type Cursor interface {
	Decode(val interface{}) error
	Err() error
	Next(ctx context.Context) bool
	Close(ctx context.Context) error
	ID() int64
	Current() []byte
}

type cursor struct {
	mongodb.Cursor
}

func (m *cursor) Current() []byte {
	return bsoncv.ToJson(m.Cursor.Current)
}

func (m *cursor) Decode(val interface{}) error {
	return json.Unmarshal(m.Current(), val)
}

func (m *cursor) Close(ctx context.Context) error {
	return m.Close(ctx)
}

type Decoder interface {
	DecodeBytes() ([]byte, error)
	Decode(val interface{}) error
	Err() error
}

type decoder struct {
	mongodb.SingleResult
}

func (m *decoder) DecodeBytes() ([]byte, error) {
	data, err := m.SingleResult.DecodeBytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode bytes")
	}
	return bsoncv.ToJson(data), nil
}

func (m *decoder) Decode(val interface{}) error {
	data, err := m.SingleResult.DecodeBytes()
	if err != nil {
		return errors.Wrap(err, "failed to decode")
	}
	return json.Unmarshal(bsoncv.ToJson(data), val)
}

type Collection struct {
	c *mongodb.Collection
}

func (c Collection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (Cursor, error) {
	cur, err := c.c.Find(ctx, filter, opts...)
	if err != nil {
		err = errors.WithStack(err)
	}
	if cur == nil {
		return nil, err
	}
	return &cursor{*cur}, err
}

func (c Collection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) (Decoder, error) {
	singleResult := c.c.FindOne(ctx, filter, opts...)
	err := singleResult.Err()
	if err != nil {
		if err == mongodb.ErrNoDocuments {
			return &decoder{*singleResult}, nil
		}
		err = errors.WithStack(err)
	}
	return &decoder{*singleResult}, err
}

func (c Collection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (Cursor, error) {
	cur, err := c.c.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		err = errors.WithStack(err)
	}
	if cur == nil {
		return nil, err
	}
	return &cursor{*cur}, err
}

func (c Collection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (string, error) {
	insertResult, err := c.c.InsertOne(ctx, document, opts...)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if id, ok := insertResult.InsertedID.(primitive.ObjectID); !ok {
		panic(fmt.Sprintf("the inserted documents ObjectID wasn't of type primitive.ObjectID %v", insertResult))
	} else {
		return id.Hex(), nil
	}
}
