package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/ponyo877/suika-shaker/internal/ui"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) IsButtonClicked(x, y int, cfg ui.ButtonConfig) bool {
	return x >= int(cfg.X) && x <= int(cfg.X+cfg.Width) &&
		y >= int(cfg.Y) && y <= int(cfg.Y+cfg.Height)
}

func (h *Handler) IsRetryButtonClicked(x, y int) bool {
	cfg := ui.NewDialogConfig()
	return x >= int(cfg.RetryX) && x <= int(cfg.RetryX+cfg.RetryWidth) &&
		y >= int(cfg.ButtonY) && y <= int(cfg.ButtonY+cfg.RetryHeight)
}

func (h *Handler) CheckMouseClick() (bool, int, int) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return true, x, y
	}
	return false, 0, 0
}

func (h *Handler) CheckTouchInput() []struct{ X, Y int } {
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	var touches []struct{ X, Y int }

	for _, id := range touchIDs {
		x, y := ebiten.TouchPosition(id)
		touches = append(touches, struct{ X, Y int }{x, y})
	}

	return touches
}
