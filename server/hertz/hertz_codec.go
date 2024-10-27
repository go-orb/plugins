package hertz

import (
	"fmt"
	"mime"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/go-orb/go-orb/codecs"
)

// TODO(jochumdev): decode body now also does content type setting, maybe separate that out

// GetContentType parses the content type from the header value.
func GetContentType(header string) (string, error) {
	ct, _, err := mime.ParseMediaType(header)
	if err != nil {
		return "", err
	}

	return ct, nil
}

// GetAcceptType parses the Accept header and checks against the available codecs
// to find a matching content type.
func GetAcceptType(ctx codecs.Map, acceptHeader string, contentType string) string {
	accept := contentType

	// If request used Form content type, return JSON instead of form.
	if accept == consts.MIMEApplicationHTMLFormUTF8 {
		accept = consts.MIMEApplicationJSONUTF8
	}

	// If explicitly asked for a specific content type, use that
	acceptSlice := strings.Split(acceptHeader, ",")
	for _, acceptType := range acceptSlice {
		ct, _, err := mime.ParseMediaType(acceptType)
		if err != nil {
			continue
		}

		// Check if we have a codec for the content type
		if _, ok := ctx[ct]; ok {
			accept = ct
		}
	}

	return accept
}

// decodeBody takes the request body and decodes it into the proto type.
func (s *Server) decodeBody(ctx *app.RequestContext, msg any) (string, error) {
	if ctx.Request.Header.IsGet() {
		return consts.MIMEApplicationJSON, ctx.BindQuery(msg)
	}

	ct := utils.FilterContentType(string(ctx.ContentType()))
	switch ct {
	case consts.MIMEApplicationJSON:
		return ct, ctx.BindJSON(msg)
	case consts.MIMEPROTOBUF:
		return ct, ctx.BindProtobuf(msg)
	case consts.MIMEApplicationHTMLForm, consts.MIMEMultipartPOSTForm:
		return consts.MIMEApplicationJSON, ctx.BindForm(msg)
	default:
		return consts.MIMETextPlain, fmt.Errorf("%w: '%s'", ErrContentTypeNotSupported, ct)
	}
}

// encodeBody takes the return proto type and encodes it into the response.
func (s *Server) encodeBody(ctx *app.RequestContext, v any) error {
	ct := utils.FilterContentType(ctx.Request.Header.Get(consts.HeaderAccept))

	switch ct {
	case consts.MIMEApplicationJSON:
		ctx.JSON(consts.StatusOK, v)
		return nil
	case consts.MIMEPROTOBUF:
		ctx.ProtoBuf(consts.StatusOK, v)
	default:
		return fmt.Errorf("%w: '%s'", ErrContentTypeNotSupported, ct)
	}

	return nil
}
