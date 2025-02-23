package renderers

import (
	"grpc-xray/models"
)

// Renderer ...
type Renderer interface {
	Render(model models.RenderModel) string
}

// GetApplicationRenderer ...
func GetApplicationRenderer() Renderer {
	return &PlainRenderer{}
}
