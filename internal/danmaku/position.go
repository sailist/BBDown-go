package danmaku

const (
	monitorWidth   = 1920
	monitorHeight  = 1080
	fontSize       = 40
	moveSpendTimeC = 8.00
	topSpendTimeC  = 4.00
	protectLength  = 50
)

// PositionController manages vertical positions for danmaku to avoid collisions.
type PositionController struct {
	moveQueue   []float64
	topQueue    []float64
	bottomQueue []float64
	maxLine     int
}

// NewPositionController creates a new PositionController with initialized queues.
func NewPositionController() *PositionController {
	maxLine := monitorHeight * protectLength / fontSize / 100
	pc := &PositionController{
		moveQueue:   make([]float64, maxLine),
		topQueue:    make([]float64, maxLine),
		bottomQueue: make([]float64, maxLine),
		maxLine:     maxLine,
	}
	return pc
}

// UpdatePosition finds an available vertical position for a danmaku.
// It returns the pixel height from the top, or -1 if no position is available.
func (pc *PositionController) UpdatePosition(danmakuMode int, time float64, length int) int {
	var vs []float64
	displayTime := topSpendTimeC

	switch danmakuMode {
	case 3:
		vs = pc.bottomQueue
	case 2:
		vs = pc.topQueue
	default:
		vs = pc.moveQueue
		displayTime = moveSpendTimeC * float64((length+5)*fontSize) / (float64(monitorWidth) + float64(length)*moveSpendTimeC)
	}

	for i := 0; i < pc.maxLine; i++ {
		if time >= vs[i] {
			vs[i] = time + displayTime
			return i * fontSize
		}
	}
	return -1
}
