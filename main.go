package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const highScoreFileName = "highscore.txt"

const (
	screenWidth  = 800
	screenHeight = 600
	sidebarWidth = 140
	ticksPerSec  = 60 // Update() is called with this frequency
	scale        = 30 // Unified scale factor for cells and sprites
)

type SpeedLevel struct {
	ticksPerDrop     int
	nextLevelTimeSec int
}

var (
	gridSize         = Size{18, 18}
	speedLevels      = []SpeedLevel{{30, 30}, {26, 60}, {22, 90}, {19, 120}, {16, 150}, {13, 180}, {11, 210}, {9, 240}, {7, 270}, {6, 300}}
	boundingBoxColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	sidebarColor     = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	backgroundColor  = color.RGBA{R: 100, G: 100, B: 100, A: 255}
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation int           // Current rotation in degrees (0, 90, 180, 270)
	size            Size          // Dimensions of the piece on the grid
	pieceType       string        // Head, Torso, Leg
	pos             Pos           // Position of the piece on the grid (top left corner)
	// dropKeyPressed  bool
}

/*
Returns scale which also considers dimensions of the image.
The size of the rendered image must be Piece.size on grid independently of the image resolution or size.
*/
func (piece *Piece) getScale() (float64, float64) {
	return scale * float64(piece.size.w) / float64(piece.image.Bounds().Max.X), scale * float64(piece.size.h) / float64(piece.image.Bounds().Max.Y)
}

func (piece *Piece) isBomb() bool {
	return piece.pieceType == "Bomb"
}

/*
dropPiece moves the active piece as far down as possible.
*/
func (g *Game) dropPiece() {
	for g.canMove(g.activePiece, 0, 1) {
		g.activePiece.pos.y++
	}
	g.handleActivePieceLanded()
}

func mustLoadImage(path string) *ebiten.Image {
	img, err := LoadImage(path)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

/*
loadTopScores loads and returns the top 5 scores from the highscore.txt file.
*/
func readScoresFromFile() []int {
	data, err := os.ReadFile(highScoreFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}
		}
		log.Printf("Failed to read high scores: %v", err)
		return []int{}
	}

	scoreStrings := strings.Split(string(data), "\n")
	var scores []int
	for _, scoreStr := range scoreStrings {
		if scoreStr == "" {
			continue
		}
		score, err := strconv.Atoi(scoreStr)
		if err == nil {
			scores = append(scores, score)
		}
	}
	return scores
}

func (g *Game) loadTopScores() []int {
	scores := readScoresFromFile()
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))
	if len(scores) > 5 {
		scores = scores[:5]
	}
	return scores
}

/*
saveScore appends the current score to the highscore.txt file.
*/
func (g *Game) saveScore(score int) {
	file, err := os.OpenFile(highScoreFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open high score file: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%d\n", score)); err != nil {
		log.Printf("Failed to write score: %v", err)
	}
}

/*
endGame handles the end of the game, saving the score and checking for a new high score.
*/
func (g *Game) endGame() {
	g.gameOver = true
	MUSIC_PLAYER.Pause()
	log.Printf("Game ended. Spawn stat: %v", g.spawnStat)
	// Save the current score to the highscore file
	g.saveScore(g.score)

	if g.score >= g.loadHighScore() {
		ebitenutil.DebugPrintAt(ebiten.NewImage(screenWidth, screenHeight), "New High Score!", screenWidth/2-50, screenHeight/2+40)
		log.Println("New high score achieved!")
	}
}

/*
loadHighScore loads the high score from a file.
*/
func (g *Game) loadHighScore() int {
	scores := readScoresFromFile()
	var highScore int
	for _, score := range scores {
		if score > highScore {
			highScore = score
		}
	}
	return highScore
}

type Game struct {
	grid                [][]*Piece // Store piece references for each grid cell
	lockedPieces        []*Piece   // Array to store locked pieces, sorted first by y then x coordinate
	activePiece         *Piece
	nextPiece           *Piece
	score               int
	frameCount          int
	dropFrameCount      int // counts frames. used for determining time to drop the piece
	gameTimeSec         float32
	gameOver            bool
	rotateKeys          []ebiten.Key
	leftKeys            []ebiten.Key
	rightKeys           []ebiten.Key
	dropKeys            []ebiten.Key
	speedKeys           []ebiten.Key
	rotateKeyPressed    bool
	moveLeftKeyPressed  bool
	moveRightKeyPressed bool
	dropKeyPressed      bool
	speedupKeyPressed   bool
	speedLevelIdx       int            // index in speedLevels
	spawnStat           map[string]int // game statistics: number of spawned pieces per piece type
}

/*
Reset reinitializes the game state to start a new game.
*/
func (g *Game) Reset() {
	log.Printf("Game reset. Spawn stat: %v", g.spawnStat)
	*g = *NewGame()
	MUSIC_PLAYER.Play()
}

/*
LoadImage loads an image from the specified file path.

Parameters:
- path: The file path to the image.

Returns:
- A pointer to the loaded ebiten.Image.
*/
func LoadImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load image from path %s: %w", path, err)
	}
	return img, nil
}

var allPieces []Piece
var allBodies []*Body

func init() {
	allPieces = []Piece{
		{image: mustLoadImage("assets/head10x10.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "Head"},
		{image: mustLoadImage("assets/torso10x10.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "Torso"},
		{image: mustLoadImage("assets/right_brk_torso10x10.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "RightBrkTorso"},
		{image: mustLoadImage("assets/left_brk_torso10x10.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "LeftBrkTorso"},
		{image: mustLoadImage("assets/leg10x10.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "Leg"},
		{image: mustLoadImage("assets/bomb11x11.png"), currentRotation: 0, size: Size{1, 1}, pieceType: "Bomb"},
	}

	// size of a piece
	genericSize := allPieces[0].size

	allBodies = []*Body{
		{ // bar shape, consists of 3 parts
			name:  "Fellow",
			score: 1000,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				{pos: Pos{0, genericSize.h}, rotation: 0, pieceType: "Torso"},
				{pos: Pos{0, 2 * genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		}, { // bar shape, consists of 2 parts
			name:  "Asshead",
			score: 500,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				{pos: Pos{0, genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
		{ // bar shape, consists of 3 parts
			name:  "Right broken",
			score: 3000,
			bodyPieces: []BodyPiece{ // defined as L shape
				{pos: Pos{0, 0}, rotation: 90, pieceType: "Head"},
				{pos: Pos{genericSize.h, 0}, rotation: 0, pieceType: "RightBrkTorso"},
				{pos: Pos{genericSize.h, genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
		{ // bar shape, consists of 3 parts
			name:  "Left broken",
			score: 3000,
			bodyPieces: []BodyPiece{ // defined as L shape
				{pos: Pos{0, 0}, rotation: 0, pieceType: "LeftBrkTorso"},
				{pos: Pos{genericSize.h, 0}, rotation: 270, pieceType: "Head"},
				{pos: Pos{0, genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
	}
}

/*
NewGame creates and returns a new Game instance with initialized pieces
and game state.
*/
func NewGame() *Game {
	MUSIC_PLAYER.Play()
	// allocate grid
	theGrid := make([][]*Piece, gridSize.w)
	for i := 0; i < gridSize.w; i++ {
		theGrid[i] = make([]*Piece, gridSize.h)
	}

	// initialze bodies
	for _, body := range allBodies {
		print("%v", body)
		body.init()
	}

	game := &Game{

		rotateKeys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpad8},
		leftKeys:   []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyNumpad7},
		rightKeys:  []ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyNumpad9},
		dropKeys:   []ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyNumpad5, ebiten.KeySpace},
		speedKeys:  []ebiten.Key{ebiten.KeyS},
		spawnStat:  make(map[string]int),
		grid:       theGrid,
	}
	game.activePiece = game.generatePiece()
	game.nextPiece = game.generatePiece()
	return game
}

/*
Update handles the game logic for each frame, including user input,
piece movement, and game state updates.

Returns:
- An error if the update fails, otherwise nil.
*/
func (g *Game) Update() error {
	g.frameCount++
	g.gameTimeSec += 1 / float32(ticksPerSec)

	g.restart() // Always check for restart

	if !g.gameOver {
		g.movePieceInDirection(-1, g.leftKeys, &g.moveLeftKeyPressed)
		g.movePieceInDirection(1, g.rightKeys, &g.moveRightKeyPressed)
		g.rotate()
		g.speedup()
		if g.checkTimeToDrop() {
			g.drop()
		}
		g.handleKeyPress(g.dropKeys, &g.dropKeyPressed, g.dropPiece)
	}

	return nil
}

/*
Restart the game if the restart button is clicked on the sidebar.
*/
func (g *Game) restart() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		sidebarX := screenWidth - sidebarWidth
		if isWithinBounds(Pos{x, y}, Size{0, 0}, Pos{sidebarX + 10, 160}, Pos{sidebarX + 110, 180}) {
			g.Reset()
		}
	}
}

/*
determines if it is time to drop. increases speed if it is time to go to the next speed level.
returns if it is time to drop.
*/
func (g *Game) checkTimeToDrop() bool {
	g.dropFrameCount++

	speedLevel := speedLevels[g.speedLevelIdx]
	if speedLevel.ticksPerDrop <= g.dropFrameCount {
		g.dropFrameCount = 0

		if g.speedLevelIdx+1 < len(speedLevels) && float32(speedLevel.nextLevelTimeSec) < g.gameTimeSec {
			g.speedLevelIdx++
			log.Printf("speed level increased to %d at %d frames, %f sec", g.speedLevelIdx, g.frameCount, g.gameTimeSec)
		}

		return true
	} else {
		return false
	}
}

/*
drop moves the active piece down the grid,
locking it in place if it cannot move further.
*/
func (g *Game) drop() {
	if !g.canMove(g.activePiece, 0, 1) {
		g.handleActivePieceLanded()
	} else {
		g.activePiece.pos.y++
	}
}

/*
handleKeyPress centralizes the handling of key presses to reduce redundancy.

Parameters:
- key: The ebiten key to check.
- pressed: A pointer to a boolean indicating if the key was previously pressed.
- action: The action to perform if the key is pressed.
*/
func (g *Game) handleKeyPress(keys []ebiten.Key, pressed *bool, action func()) {
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			if !*pressed {
				action()
			}
			*pressed = true
			return
		}
	}
	*pressed = false
}

/*
rotate handles the rotation of the active piece when the space key is pressed.
*/
func (g *Game) rotate() {
	g.handleKeyPress(g.rotateKeys, &g.rotateKeyPressed, func() {
		if !g.activePiece.isBomb() { // do not rotate bomb (it is symmetric and has a visual sparkle)
			g.activePiece.currentRotation = (g.activePiece.currentRotation + 90) % 360
		}
	})
}

/*
speedup handles the speding up when the "increase speed" key is pressed.
*/
func (g *Game) speedup() {
	g.handleKeyPress(g.speedKeys, &g.speedupKeyPressed, func() {
		if g.speedLevelIdx+1 < len(speedLevels) {
			g.speedLevelIdx++
			log.Printf("speed level increased manually to %d at %f sec", g.speedLevelIdx, g.gameTimeSec)
		}
	})
}

/*
movePieceInDirection moves the active piece one cell in the specified direction
when the corresponding arrow key is pressed.

Parameters:
- direction: The direction to move the piece (-1 for left, 1 for right).
- key: The ebiten key to check for the direction.
- pressed: A pointer to a boolean indicating if the key was previously pressed.
*/
func (g *Game) movePieceInDirection(direction int, keys []ebiten.Key, pressed *bool) {
	g.handleKeyPress(keys, pressed, func() {
		if g.canMove(g.activePiece, direction, 0) {
			g.activePiece.pos.x += direction
		}
	})
}

/*
Draw renders the current game state to the screen, including the active piece,
locked pieces, and sidebar information.

Parameters:
- screen: The ebiten.Image to draw the game state onto.
*/
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill only the game area with the background color
	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth-sidebarWidth), float32(screenHeight), backgroundColor, false)

	if g.gameOver {
		if g.score >= g.loadHighScore() {
			ebitenutil.DebugPrintAt(screen, "New High Score!", screenWidth/2-50, screenHeight/2)
		} else {
			ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-50, screenHeight/2)
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", g.score), screenWidth/2-50, screenHeight/2+20)
	}

	sidebarX := screenWidth - sidebarWidth
	vector.DrawFilledRect(screen, float32(sidebarX), 0, sidebarWidth, screenHeight, sidebarColor, false)
	drawBorder(screen)
	g.drawLockedPieces(screen)
	g.drawBoundingBox(screen)

	op := &ebiten.DrawImageOptions{}
	g.applyRotationToPiece(op, g.activePiece)
	screen.DrawImage(g.activePiece.image, op)
	g.drawSidebar(screen)
}

/*
drawSidebar renders the sidebar, including the next piece, restart button,
and score.

Parameters:
- screen: The ebiten.Image to draw the sidebar onto.
*/
func (g *Game) drawSidebar(screen *ebiten.Image) {
	sidebarX := screenWidth - sidebarWidth

	// Draw "Next Piece"
	ebitenutil.DebugPrintAt(screen, "NEXT PIECE", sidebarX+10, 20)
	op := &ebiten.DrawImageOptions{}
	imageScaleX, imageScaleY := g.nextPiece.getScale()
	op.GeoM.Scale(imageScaleX, imageScaleY) // Apply scaling to the next piece
	op.GeoM.Translate(float64(sidebarX+40), 50)
	screen.DrawImage(g.nextPiece.image, op)

	// Draw restart button
	ebitenutil.DebugPrintAt(screen, "RESTART", sidebarX+10, 160)

	// Draw top 5 scores
	ebitenutil.DebugPrintAt(screen, "TOP 5 SCORES", sidebarX+10, 200)
	topScores := g.loadTopScores()
	for i, score := range topScores {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d: %d", i+1, score), sidebarX+10, 220+i*20)
	}

	// Draw current score
	ebitenutil.DebugPrintAt(screen, "SCORE", sidebarX+10, 120)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", g.score), sidebarX+80, 120)

	// Draw current speed level
	ebitenutil.DebugPrintAt(screen, "SPEED", sidebarX+10, 140)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", g.speedLevelIdx+1), sidebarX+80, 140)

	// Draw hints about joint bodies
	hintPosLL := Pos{sidebarX, screenHeight}
	hintRowHeight := 0
	for i := 0; i < len(allBodies); i++ {
		body := allBodies[len(allBodies)-1-i]

		ok, hintAreaSize := g.drawSidebarHint(screen, body, hintPosLL)

		// go to row above if no more space on the sidebar row
		if !ok {
			hintPosLL.x = sidebarX
			hintPosLL.y -= hintRowHeight + 10
			_, hintAreaSize = g.drawSidebarHint(screen, body, hintPosLL)
		}

		if hintRowHeight < hintAreaSize.h {
			hintRowHeight = hintAreaSize.h
		}

		hintPosLL.x += hintAreaSize.w
	}
}

func (g *Game) drawSidebarHint(screen *ebiten.Image, body *Body, posLL Pos) (bool, Size) {
	hintTextAreaHeight := 40
	hintAreaSize := Size{70, hintTextAreaHeight} // text + pieces together

	// check if outside of screen
	if screenWidth < posLL.x+hintAreaSize.w {
		return false, hintAreaSize
	}

	// draw text
	hintTextPos := addPos(posLL, Pos{0, -hintTextAreaHeight})
	ebitenutil.DebugPrintAt(screen, body.name, hintTextPos.x, hintTextPos.y)                        // todo: render text at the center of the hint area
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", body.score), hintTextPos.x, hintTextPos.y+20) // todo: render text at the center of the hint area

	// get dimension of the body
	boxPos, boxSize := g.getBoundingBox(body)
	hintAreaSize.h += boxSize.h * scale

	// draw text pieces
	bodyPosUL := addPos(posLL, Pos{hintAreaSize.w/2 - scale*boxSize.w/2, -hintAreaSize.h})
	for _, bp := range body.bodyPieces {
		piece := g.getPiece(bp.pieceType)
		w, h := grid2ScrSize(float32(piece.size.w)/2, float32(piece.size.h)/2)

		op := &ebiten.DrawImageOptions{}
		imageScaleX, imageScaleY := piece.getScale()
		op.GeoM.Scale(imageScaleX, imageScaleY) // Apply scaling to the next piece
		op.GeoM.Translate(float64(-w), float64(-h))
		op.GeoM.Rotate(-getRotationTheta(bp.rotation))
		op.GeoM.Translate(float64(bodyPosUL.x+(bp.pos.x-boxPos.x)*scale), float64(bodyPosUL.y+(bp.pos.y-boxPos.y)*scale))
		op.GeoM.Translate(float64(w), float64(h))
		screen.DrawImage(piece.image, op)
	}

	return true, hintAreaSize
}

/*
Returns the bounding box of a body in bodyPiece CS.
*/
func (g *Game) getBoundingBox(body *Body) (Pos, Size) {
	minPos := Pos{}
	maxPos := Pos{}

	for i, bp := range body.bodyPieces {
		piece := g.getPiece(bp.pieceType)
		rotatedSize := rotateSize(piece.size, piece.currentRotation)

		if i == 0 || bp.pos.x < minPos.x {
			minPos.x = bp.pos.x
		}
		if i == 0 || bp.pos.y < minPos.y {
			minPos.y = bp.pos.y
		}
		if i == 0 || maxPos.x < bp.pos.x+rotatedSize.w {
			maxPos.x = bp.pos.x + rotatedSize.w
		}
		if i == 0 || maxPos.y < bp.pos.y+rotatedSize.h {
			maxPos.y = bp.pos.y + rotatedSize.h
		}
	}

	return minPos, Size{maxPos.x - minPos.x, maxPos.y - minPos.y}
}

func (g *Game) getPiece(pieceType string) *Piece {
	idx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == pieceType })
	return &allPieces[idx]
}

/*
drawLockedPieces renders all locked pieces on the grid.

Parameters:
- screen: The ebiten.Image to draw the locked pieces onto.
*/
func (g *Game) drawLockedPieces(screen *ebiten.Image) {
	for _, lp := range g.lockedPieces {
		op := &ebiten.DrawImageOptions{}

		// Calculate the top-left corner of the locked piece in screen coordinates.

		g.applyRotationToPiece(op, lp)
		screen.DrawImage(lp.image, op)
	}
}

/*
applyRotationToPiece applies the current rotation to a piece and prepares it
for drawing on the screen.

Parameters:
- op: The ebiten.DrawImageOptions to apply transformations.
- piece: The Piece to apply the rotation to.
*/
func (g *Game) applyRotationToPiece(op *ebiten.DrawImageOptions, piece *Piece) {
	imageScaleX, imageScaleY := piece.getScale()
	op.GeoM.Scale(imageScaleX, imageScaleY)

	// Center the rotation point (relative to the piece).
	x, y := grid2ScrPos(float32(piece.pos.x), float32(piece.pos.y))
	w, h := grid2ScrSize(float32(piece.size.w)/2, float32(piece.size.h)/2)
	centerX, centerY := x+w, y+h

	// Translate to the center of the piece.
	op.GeoM.Translate(float64(-w), float64(-h))

	// Rotate around the center.
	op.GeoM.Rotate(-getRotationTheta(piece.currentRotation))

	// Translate the piece back to its grid position.
	op.GeoM.Translate(float64(centerX), float64(centerY))
}

/*
drawBoundingBox draws a bounding box around the active piece for visual
reference.

Parameters:
- screen: The ebiten.Image to draw the bounding box onto.
*/
func (g *Game) drawBoundingBox(screen *ebiten.Image) {
	x, y := grid2ScrPos(float32(g.activePiece.pos.x), float32(g.activePiece.pos.y))
	w, h := grid2ScrSize(float32(g.activePiece.size.w), float32(g.activePiece.size.h))
	vector.StrokeRect(screen, x, y, w+1, h+1, 1, boundingBoxColor, false)
}

/*
drawBorder draws a border around the game area.

Parameters:
- screen: The ebiten.Image to draw the border onto.
*/
func drawBorder(screen *ebiten.Image) {
	x, y := grid2ScrPos(0.5, -0.5)
	w, h := grid2ScrSize(float32(gridSize.w-1), float32(gridSize.h))
	// draw a rectangle with thick border. the top border is invisible (intentionally outside of the screen) intentionally.
	vector.StrokeRect(screen, x, y, w, h, scale, boundingBoxColor, false)
}

/*
Layout returns the logical screen dimensions for the game.

Parameters:
- outsideWidth: The width of the outside environment.
- outsideHeight: The height of the outside environment.

Returns:
- The width and height of the game screen.
*/
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

/*
canMove checks if the active piece can move to a new position on the grid.

Parameters:
- dx: The change in the x-direction.
- dy: The change in the y-direction.

Returns:
- True if the piece can move to the new position, otherwise false.
*/
func (g *Game) canMove(piece *Piece, dx, dy int) bool {
	newPos := Pos{piece.pos.x + dx, piece.pos.y + dy}
	size := rotateSize(piece.size, piece.currentRotation)

	if !isWithinBounds(newPos, size, Pos{1, 1}, Pos{gridSize.w - 1, gridSize.h - 1}) {
		return false
	}

	for _, piece := range g.lockedPieces {
		if piece.isColliding(newPos, size) {
			return false
		}
	}

	return true
}

/*
Call this when the active piece is landed. If does the following:
If the active piece is a bomb: destroys piece below.
Otherwise locks the piece, join and score bodies, then spawn a new piece.
Spawn a new piece.
*/
func (g *Game) handleActivePieceLanded() {
	if g.activePiece.isBomb() {
		below := addPos(g.activePiece.pos, Pos{0, 1})
		// is the location below the bomb within the grid?
		if isWithinBounds(below, Size{1, 1}, Pos{1, 1}, Pos{gridSize.w - 1, gridSize.h - 1}) {
			// remove (unlock) each piece below the bomb
			for i := 0; i < g.activePiece.size.w; i++ {
				piece := g.grid[below.x+i][below.y]
				if piece != nil {
					g.unlockPiece(piece)
				}
			}
		}
	} else {
		g.lockPiece(g.activePiece)
		g.joinAndScorePieces([]*Piece{g.activePiece})
	}
	g.spawnNewPiece()
}

/*
lockPiece locks the active piece in its current position on the grid,
adding it to the list of locked pieces.
*/
func (g *Game) lockPiece(piece *Piece) {
	// find in the sorted locked list
	idx := sort.Search(len(g.lockedPieces), func(i int) bool {
		return piece.pos.y < g.lockedPieces[i].pos.y || (piece.pos.y == g.lockedPieces[i].pos.y && piece.pos.x <= g.lockedPieces[i].pos.x)
	})

	if idx < len(g.lockedPieces) && g.lockedPieces[idx] == piece {
		log.Fatalf("The piece %v is not expected in the locked list!", piece)
	}

	// insert to sorted list
	g.lockedPieces = append(g.lockedPieces, nil)
	copy(g.lockedPieces[idx+1:], g.lockedPieces[idx:])
	g.lockedPieces[idx] = piece

	// add references to the locked piece in the grid
	g.changePieceInGrid(piece, true)
}

/*
Remove piece from the locked list and grid matrix.
*/
func (g *Game) unlockPiece(lockedPiece *Piece) {
	// find in the sorted locked list
	idx := sort.Search(len(g.lockedPieces), func(i int) bool {
		return lockedPiece.pos.y < g.lockedPieces[i].pos.y || (lockedPiece.pos.y == g.lockedPieces[i].pos.y && lockedPiece.pos.x <= g.lockedPieces[i].pos.x)
	})

	if idx == len(g.lockedPieces) || g.lockedPieces[idx] != lockedPiece {
		log.Fatalf("The piece %v is expected in the locked list!", lockedPiece)
	}

	// remove from sorted list
	g.lockedPieces = append(g.lockedPieces[:idx], g.lockedPieces[idx+1:]...)

	// remove references to the locked piece in the grid
	g.changePieceInGrid(lockedPiece, false)
}

/*
add/remove references to the locked piece in the grid
*/
func (g *Game) changePieceInGrid(piece *Piece, add bool) {
	rotatedSize := rotateSize(piece.size, piece.currentRotation)
	for x := piece.pos.x; x < piece.pos.x+rotatedSize.w; x++ {
		for y := piece.pos.y; y < piece.pos.y+rotatedSize.h; y++ {
			if add {
				g.grid[x][y] = piece
			} else {
				g.grid[x][y] = nil
			}
		}
	}
}

/*
spawnNewPiece make the next piece to be the active piece and
creates the next active piece from the available pieces.
*/
func (g *Game) spawnNewPiece() {
	if g.activePiece.pos.y == 0 && !g.canMove(g.activePiece, 0, 1) {
		g.endGame()
		return
	}

	g.activePiece = g.nextPiece
	g.nextPiece = g.generatePiece()
}

func (g *Game) joinAndScorePieces(pieces []*Piece) {
	log.Printf("joinAndScorePieces(pieces: %v)", pieces)

	joinedCnt := 0
	for 0 < len(pieces) {
		piece := pieces[0]
		pieces = pieces[1:]

		if piece != nil {
			for _, body := range allBodies {
				posList := body.matchAtLockedPiece(g, piece)

				if posList != nil {
					g.score += body.score
					g.removePieces(posList)
					joinedCnt++
				}
			}
		}
	}

	if 0 < joinedCnt {
		pieces = g.compactGrid()

		// if any pieces has fallen => recurse
		if pieces != nil {
			g.joinAndScorePieces(pieces)
		}
	}
}

func (g *Game) removePieces(positions []Pos) {
	for _, pos := range positions {
		piece := g.grid[pos.x][pos.y]
		g.unlockPiece(piece)
	}
}

func (g *Game) compactGrid() []*Piece {
	fallenPieces := make([]*Piece, 0)

	for i := len(g.lockedPieces) - 1; 0 <= i; i-- {
		piece := g.lockedPieces[i]

		// check if piece can fall
		size := rotateSize(piece.size, piece.currentRotation)
		dy := size.h - 1
		for g.canMove(piece, 0, dy+1) {
			dy++
		}

		if size.h <= dy {
			log.Printf("Moving piece '%s'@%v down by %d", piece.pieceType, piece.pos, dy)
			g.unlockPiece(piece)
			piece.pos.y += dy
			g.lockPiece(piece)

			fallenPieces = append(fallenPieces, piece)
		}
	}

	return fallenPieces
}

/*
generatePiece creates a new piece from the available pieces and
positions it at the top of the grid.
*/
func (g *Game) generatePiece() *Piece {
	bombIdx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.isBomb() })
	bombRelProb := float32(0.75) // relative probability to ordinary pieces

	// generate random index of the new piece. take care that the bomb has different probability than ordinary pieces
	newPieceIdx := rand.Intn(len(allPieces))
	if newPieceIdx == bombIdx && bombRelProb < rand.Float32() {
		newPieceIdx = rand.Intn(len(allPieces) - 1)
		if newPieceIdx == bombIdx {
			newPieceIdx++
		}
	}

	newPiece := allPieces[newPieceIdx]
	newPiece.pos.x = gridSize.w / 2
	newPiece.pos.y = 0
	if !newPiece.isBomb() { // do not rotate bomb (it is symmetric and has a visual sparkle)
		newPiece.currentRotation = rand.Intn(4) * 90
	}

	// update statistics
	val := g.spawnStat[newPiece.pieceType]
	val++
	g.spawnStat[newPiece.pieceType] = val

	return &newPiece
}

/*
main initializes the game window and starts the game loop.
*/
var MUSIC_PLAYER *Audio

func init() {

	MUSIC_PLAYER = NewAudio("assets/theme.mp3")
}

func main() {
	log.SetFlags(log.Ltime)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("TESTRis - Fixed Piece Spawning and Locking")
	// init() is already called automatically by Go runtime
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
