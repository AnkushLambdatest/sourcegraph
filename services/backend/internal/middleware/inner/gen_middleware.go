// +build ignore

package main

import (
	"text/template"

	"sourcegraph.com/sourcegraph/sourcegraph/pkg/gen"
)

func main() {
	svcs := []string{
		"../../../../../api/sourcegraph/sourcegraph.pb.go",
		"../../../../../vendor/sourcegraph.com/sourcegraph/srclib/store/pb/srcstore.pb.go",
	}
	gen.Generate("inner_middleware.go", tmpl, svcs, nil, "")
}

var tmpl = template.Must(template.New("").Delims("<<<", ">>>").Parse(`// GENERATED CODE - DO NOT EDIT!
// @` + "generated" + `
//
// Generated by:
//
//   go run gen_middleware.go
//
// Called via:
//
//   go generate
//

package inner

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"
	"sourcegraph.com/sourcegraph/srclib/store/pb"
	"sourcegraph.com/sqs/pbtypes"
	"sourcegraph.com/sourcegraph/sourcegraph/api/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/errcode"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/inventory"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend/internal/middleware/inner/trace"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend"
	"sourcegraph.com/sourcegraph/sourcegraph/services/svc"
)

// Services returns the local services wrapped with auth, etc.
func Services() svc.Services {
	return svc.Services{
		<<<range .>>>
			<<<.Name>>>: wrapped<<<.Name>>>{},
		<<<end>>>
	}
}

<<<range .>>>
	type wrapped<<<.Name>>> struct{}
  <<<$service := .>>>
	<<<range .Methods>>>
		func (s wrapped<<<$service.Name>>>) <<<.Name>>>(ctx context.Context, param *<<<.ParamType>>>) (res *<<<.ResultType>>>, err error) {
			start := time.Now()
			ctx = trace.Before(ctx, "<<<$service.Name>>>", "<<<.Name>>>", param)
			defer func(){
		  	trace.After(ctx, "<<<$service.Name>>>", "<<<.Name>>>", param, err, time.Since(start))
			}()
			res, err = backend.Services.<<<$service.Name>>>.<<<.Name>>>(ctx, param)
			if res == nil && err == nil {
				err = grpc.Errorf(codes.Internal, "<<<$service.Name>>>.<<<.Name>>> returned nil, nil")
			}
			if err != nil && !DebugMode(ctx) {
				if code := errcode.GRPC(err); code == codes.Unknown || code == codes.Internal {
					// Sanitize, because these errors should not be user visible.
					err = grpc.Errorf(code, "<<<$service.Name>>>.<<<.Name>>> failed with internal error.")
				}
			}
			return
		}
	<<<end>>>
<<<end>>>
`))
