package form

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/go-orb/go-orb/codecs"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-orb/plugins/codecs/form/testdata"
)

type LoginRequest struct {
	Username string `json:"username,omitempty" form:"username,omitempty"`
	Password string `json:"password,omitempty" form:"password,omitempty"`
}

type TestModel struct {
	ID   int32  `json:"id"   form:"id"`
	Name string `json:"name" form:"name"`
}

var marshalTests = []struct {
	Input    any
	Expected string
}{
	{
		Input: LoginRequest{
			Username: "micro",
			Password: "micro_pwd",
		},
		Expected: "password=micro_pwd&username=micro",
	},
	{
		Input: LoginRequest{
			Username: "micro",
			Password: "",
		},
		Expected: "username=micro",
	},
	{
		Input: TestModel{
			ID:   1,
			Name: "micro",
		},
		Expected: "id=1&name=micro",
	},
}

func TestFormCodecMarshal(t *testing.T) {
	form := getCodec(t)

	for i, test := range marshalTests {
		t.Run("MarshalTest"+strconv.Itoa(i), func(t *testing.T) {
			c, err := form.Encode(&test.Input)
			require.NoError(t, err)

			content := string(c)
			assert.Equal(t, test.Expected, content)
		})
	}
}

func TestFormCodecUnmarshal(t *testing.T) {
	req := &LoginRequest{
		Username: "micro",
		Password: "micro_pwd",
	}
	content, err := getCodec(t).Encode(req)
	require.NoError(t, err)

	bindReq := new(LoginRequest)
	require.NoError(t, getCodec(t).Decode(content, bindReq))
	assert.Equal(t, "micro", bindReq.Username)
	assert.Equal(t, "micro_pwd", bindReq.Password)
}

func TestProtoEncodeDecode(t *testing.T) {
	in := testdata.Complex{
		Id:      2233,
		NoOne:   "2233",
		Simple:  &testdata.Simple{Component: "5566"},
		Simples: []string{"3344", "5566"},
		B:       true,
		Sex:     testdata.Sex_woman,
		Age:     18,
		A:       19,
		Count:   3,
		Price:   11.23,
		D:       22.22,
		Byte:    []byte("123"),
		Map:     map[string]string{"micro": "https://go-micro.dev/"},

		Timestamp: &timestamppb.Timestamp{Seconds: 20, Nanos: 2},
		Duration:  &durationpb.Duration{Seconds: 120, Nanos: 22},
		Field:     &fieldmaskpb.FieldMask{Paths: []string{"1", "2"}},
		Double:    &wrapperspb.DoubleValue{Value: 12.33},
		Float:     &wrapperspb.FloatValue{Value: 12.34},
		Int64:     &wrapperspb.Int64Value{Value: 64},
		Int32:     &wrapperspb.Int32Value{Value: 32},
		Uint64:    &wrapperspb.UInt64Value{Value: 64},
		Uint32:    &wrapperspb.UInt32Value{Value: 32},
		Bool:      &wrapperspb.BoolValue{Value: false},
		String_:   &wrapperspb.StringValue{Value: "go-micro"},
		Bytes:     &wrapperspb.BytesValue{Value: []byte("123")},
	}
	content, err := getCodec(t).Encode(&in)
	require.NoError(t, err)

	expected := "a=19&age=18&b=true&bool=false&byte=MTIz&bytes=MTIz&count=3&d=" +
		"22.22&double=12.33&duration=2m0.000000022s&field=1%2C2&float=12.34&id=" +
		"2233&int32=32&int64=64&map%5Bmicro%5D=https%3A%2F%2Fgo-micro.dev%2F&" +
		"numberOne=2233&price=11.23&sex=woman&simples=3344&simples=5566&string=go-micro" +
		"&timestamp=1970-01-01T00%3A00%3A20.000000002Z&uint32=32&uint64=64&very_simple.component=5566"

	assert.Equal(t, expected, string(content))

	in2 := testdata.Complex{}
	require.NoError(t, getCodec(t).Decode(content, &in2))
	assert.Equal(t, int64(2233), in2.GetId())
	assert.Equal(t, "2233", in2.GetNoOne())
	assert.NotNil(t, in2.GetSimple().GetComponent())
	assert.Equal(t, "5566", in2.GetSimple().GetComponent())
	assert.NotNil(t, in2.GetSimples())
	assert.Len(t, in2.GetSimples(), 2)
	assert.Equal(t, "3344", in2.GetSimples()[0])
	assert.Equal(t, "5566", in2.GetSimples()[1])
}

func TestDecodeStructPb(t *testing.T) {
	req := new(testdata.StructPb)
	query := `data={"name":"micro"}&data_list={"name1": "micro"}&data_list={"name2": "go-micro"}`
	require.NoError(t, getCodec(t).Decode([]byte(query), req))
	assert.Equal(t, "micro", req.GetData().GetFields()["name"].GetStringValue())
	assert.Len(t, req.GetDataList(), 2)
	assert.Equal(t, "micro", req.GetDataList()[0].GetFields()["name1"].GetStringValue())
	assert.Equal(t, "go-micro", req.GetDataList()[1].GetFields()["name2"].GetStringValue())
}

func TestDecodeBytesValuePb(t *testing.T) {
	url := "https://example.com/xx/?a=1&b=2&c=3"
	val := base64.URLEncoding.EncodeToString([]byte(url))
	content := "bytes=" + val
	in2 := &testdata.Complex{}
	require.NoError(t, getCodec(t).Decode([]byte(content), in2))
	assert.Equal(t, url, string(in2.GetBytes().GetValue()))
}

func TestEncodeFieldMask(t *testing.T) {
	req := &testdata.HelloRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"foo", "bar"}},
	}
	assert.Equal(t, "updateMask=foo,bar", EncodeFieldMask(req.ProtoReflect()))
}

func TestOptional(t *testing.T) {
	v := int32(100)
	req := &testdata.HelloRequest{
		Name:     "foo",
		Sub:      &testdata.Sub{Name: "bar"},
		OptInt32: &v,
	}
	e, err := getCodec(t).EncodeValues(req)
	require.NoError(t, err)
	assert.Equal(t, "name=foo&optInt32=100&sub.naming=bar", e.Encode())
}

func getCodec(t *testing.T) *Form {
	codec, err := codecs.Plugins.Get(Name)
	require.NoError(t, err)

	form, ok := codec.(*Form)
	assert.True(t, ok)

	return form
}
