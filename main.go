package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const highScoreFileName = "highscore.txt"

const (
	screenWidth  = 800
	screenHeight = 600
	sidebarWidth = 140
	ticksPerSec  = 60 // Update() is called with this frequency
	scale        = 30 // Unified scale factor for cells and sprites
	
	DrawOrderBkgd = 10
	DrawOrderWaveEffect = 15
	DrawOrderGrid = 20
	DrawOrderActivePiece = 30
	DrawOrderSideBar = 40
	DrawOrderGameOver = 50
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
	waveEffectColor       = color.RGBA{R: 183, G: 87, B: 8, A: 255}
	waveEffectLifeTimeSec = float32(0.5) // length of the effect
	waveEffectFillPcnt    = 0.3 // means x percent of the effect area is filled with the wave
	userInput        *UserInput
)

/*
dropPiece moves the active piece as far down as possible.
*/
func (g *Game) dropPiece() {
	g.grid.drop(g.activePiece)
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
	g.apc.activate(false)
	MUSIC_PLAYER.Pause()
	log.Printf("Game ended. Spawn stat: %v", g.spawnStat)
	// Save the current score to the highscore file
	g.saveScore(g.score)

	gameOverText := []string{}
	if g.score >= g.loadHighScore() {
		gameOverText = append(gameOverText, "New High Score!")
		log.Printf("New high score %d achieved!", g.score)
	} else {
		gameOverText = append(gameOverText, "GAME OVER")
	}
	gameOverText = append(gameOverText, fmt.Sprintf("Score: %d", g.score))

	g.gameOver.text = gameOverText
	g.gameOver.activate(true)
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
	compMgr             *ComponentMgr
	background          *BackgroundComp
	wave                *WaveEffectComp
	grid                *GridComp
	input               *UserInput
	apc                 *PieceComp
	gameOver            *DialogComp
	sideBar             *SideBarComp
	activePiece         *Piece
	nextPiece           *Piece
	score               int
	frameCount          int
	dropFrameCount      int // counts frames. used for determining time to drop the piece
	gameTimeSec         float32
	speedLevelIdx       int                // index in speedLevels
	spawnProb           map[string]float32 // relative probability by piece type (default is 1.0)
	spawnStat           map[string]int     // game statistics: number of spawned pieces per piece type
}

/*
Reset reinitializes the game state to start a new game.
*/
func (g *Game) Reset() {
	log.Printf("Game reset. Spawn stat: %v", g.spawnStat)

	g.compMgr.reset() // makes all component inactive

	g.activePiece = g.generatePiece()
	g.nextPiece = g.generatePiece()
	g.score = 0
	g.frameCount = 0
	g.dropFrameCount = 0
	g.gameTimeSec = 0
	g.speedLevelIdx = 0
	g.spawnStat = map[string]int{}

	g.background.activate(true)
	g.apc.activate(true)
	g.grid.activate(true)
	g.sideBar.activate(true)
	g.apc.p = g.activePiece

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
		{ // bar shape, consists of 2 parts
			name:  "Asshead",
			score: 500,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				{pos: Pos{0, genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
		{ // bar shape, consists of 3 parts
			name:  "Fellow",
			score: 1000,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				{pos: Pos{0, genericSize.h}, rotation: 0, pieceType: "Torso"},
				{pos: Pos{0, 2 * genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
		{ // L shape, consists of 3 parts
			name:  "Killed Bill",
			score: 3000,
			bodyPieces: []BodyPiece{ // defined as L shape
				{pos: Pos{0, 0}, rotation: 90, pieceType: "Head"},
				{pos: Pos{genericSize.h, 0}, rotation: 0, pieceType: "RightBrkTorso"},
				{pos: Pos{genericSize.h, genericSize.h}, rotation: 0, pieceType: "Leg"},
			},
		},
		{ // L shape, consists of 3 parts
			name:  "Failed Yoga",
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
	// initialze bodies
	for _, body := range allBodies {
		print("%v", body)
		body.init()
	}

	game := &Game{
		compMgr:    NewComponentMgr(),
		spawnProb:  map[string]float32{ "Torso":0.5, "RightBrkTorso":0.5, "LeftBrkTorso":0.5, "Bomb":0.75 },
		spawnStat:  make(map[string]int),
	}

	if userInput == nil {
		userInput = NewUserInput(&map[string]KeyList{
			"rotate": []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpad8},
			"left": []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyNumpad7},
			"right": []ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyNumpad9},
			"drop": []ebiten.Key{ebiten.KeyArrowDown, ebiten.KeyNumpad5, ebiten.KeySpace},
			"speedup": []ebiten.Key{ebiten.KeyS}, } )
	}

	game.input = userInput
	game.background = NewBackground(Pos{0, 0}, Size{screenWidth - sidebarWidth, screenHeight}, DrawOrderBkgd)
	game.wave = NewWaveEffect(false, Rect{Pos{0, 0}, Size{screenWidth, screenHeight}}, scale, waveEffectFillPcnt, (int)(waveEffectLifeTimeSec * ticksPerSec), DrawOrderWaveEffect)
	game.grid = NewGridComp(gridSize, DrawOrderGrid)
	game.apc = NewPieceComp(game.grid, userInput, DrawOrderActivePiece)
	game.gameOver = NewModalDialog([]string{}, Pos{screenWidth/2-50, screenHeight/2}, DrawOrderGameOver)
	game.sideBar = NewSideBar(userInput, Pos{screenWidth - sidebarWidth, 0}, Size{sidebarWidth, screenHeight}, func() { game.Reset() }, DrawOrderSideBar)

	game.compMgr.add(game.background)
	game.compMgr.add(game.wave)
	game.compMgr.add(game.grid)
	game.compMgr.add(game.apc)
	game.compMgr.add(game.gameOver)
	game.compMgr.add(game.sideBar)

	game.activePiece = game.generatePiece()
	game.nextPiece = game.generatePiece()

	game.background.activate(true)
	game.apc.activate(true)
	game.grid.activate(true)
	game.sideBar.activate(true)
	game.apc.p = game.activePiece
	
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

	g.input.handleKeys()
	g.input.handleMouse()
	g.compMgr.update(g.frameCount)

	if g.gameOver.getState() == StateInactive {
		g.speedup()

		if g.checkTimeToMoveDown() {
			g.moveDown()
		}

		if g.input.isKeyPressed("drop") {
			g.dropPiece()
		}
	}

	g.sideBar.setValues(g.nextPiece, g.score, g.speedLevelIdx+1, g.loadTopScores())

	return nil
}

/*
determines if it is time to drop. increases speed if it is time to go to the next speed level.
returns if it is time to drop.
*/
func (g *Game) checkTimeToMoveDown() bool {
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
func (g *Game) moveDown() {
	if !g.grid.canMove(g.activePiece, 0, 1) {
		g.handleActivePieceLanded()
	} else {
		g.activePiece.pos.y++
	}
}

/*
speedup handles the speding up when the "increase speed" key is pressed.
*/
func (g *Game) speedup() {
	if g.input.isKeyPressed("speedup") && g.speedLevelIdx+1 < len(speedLevels) {
		g.speedLevelIdx++
		log.Printf("speed level increased manually to %d at %f sec", g.speedLevelIdx, g.gameTimeSec)
	}
}

/*
Draw renders the current game state to the screen, including the active piece,
locked pieces, and sidebar information.

Parameters:
- screen: The ebiten.Image to draw the game state onto.
*/
func (g *Game) Draw(screen *ebiten.Image) {
	g.compMgr.draw(screen)
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
Call this when the active piece is landed. If does the following:
If the active piece is a bomb: destroys piece below.
Otherwise locks the piece, join and score bodies, then spawn a new piece.
Spawn a new piece.
*/
func (g *Game) handleActivePieceLanded() {
	if g.activePiece.isBomb() {
		piecesBelow := g.grid.getPiecesBelow(g.activePiece)
		for _, piece := range piecesBelow {
			g.grid.unlockPiece(piece)
		}

		// play wave effect centered on the bomb
		x, y := grid2ScrPos(float32(g.activePiece.pos.x), float32(g.activePiece.pos.y))
		w, h := grid2ScrSize(float32(g.activePiece.size.w), float32(g.activePiece.size.h))
		g.wave.setCenter(Pos{int(x+w/2), int(y+h/2)})
		g.wave.activate(true)
	} else {
		g.grid.lockPiece(g.activePiece)
		g.joinAndScorePieces([]*Piece{g.activePiece})
	}
	g.spawnNewPiece()
}

/*
spawnNewPiece make the next piece to be the active piece and
creates the next active piece from the available pieces.
*/
func (g *Game) spawnNewPiece() {
	if g.activePiece.pos.y == 0 && !g.grid.canMove(g.activePiece, 0, 1) {
		g.endGame()
		return
	}

	g.activePiece = g.nextPiece
	g.apc.p = g.activePiece
	g.nextPiece = g.generatePiece()
}

func (g *Game) joinAndScorePieces(pieces []*Piece) {
	log.Printf("joinAndScorePieces(pieces: %v)", pieces)

	bodies := g.grid.joinPieces(pieces)

	for _, b := range bodies {
		g.score += b.score
	}

	if 0 < len(bodies) {
		pieces = g.grid.compactGrid()

		// if any piece has fallen => recurse
		if pieces != nil {
			g.joinAndScorePieces(pieces)
		}
	}
}

/*
generatePiece creates a new piece from the available pieces and
positions it at the top of the grid.
*/
func (g *Game) generatePiece() *Piece {
	// determine the random range
	var randRange float32 = 0.0
	for _, p := range allPieces {
		prob, ok := g.spawnProb[p.pieceType]
		if !ok {
			g.spawnProb[p.pieceType] = 1
			prob = 1
		}

		randRange += prob
	}

	randNum := rand.Float32() * randRange

	newPieceIdx := -1
	for newPieceIdx+1 < len(allPieces) && 0 <= randNum {
		newPieceIdx++
		randNum -= g.spawnProb[allPieces[newPieceIdx].pieceType]
	}

	newPiece := allPieces[newPieceIdx]
	newPiece.pos.x = g.grid.size.w / 2
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
