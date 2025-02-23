package renderers

import (
	"fmt"

	"grpc-xray/models"
)

// PlainRenderer ...
type PlainRenderer struct{}

// Render renders model
func (PlainRenderer) Render(model models.RenderModel) string {
	return fmt.Sprintf(
		"%s:%s -> %s:%s %s %+v",
		model.GetSrcHost(),
		model.GetSrcPort(),
		model.GetDstHost(),
		model.GetDstPort(),
		model.GetPath(),
		model.GetBody(),
	)
}
