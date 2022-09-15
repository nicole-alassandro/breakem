// Copyright (C) 2022  Toni Lassandro

// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.

// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.

// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package breakem

import (
	"image"
	"image/color"
	"math/rand"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Brick struct {
	Rect   image.Rectangle
	Points uint8
}

type Ball struct {
	Rect image.Rectangle
	Vel  image.Point
}

type Paddle struct {
	Rect image.Rectangle
	Vel  int
}

type Game struct {
	Lives  int
	Score  uint8
	Paddle Paddle
	Ball   Ball
	Bricks []Brick

	Audio struct {
		Context  *audio.Context
		Tone     Tone
		NextTone int
		Stopper  <-chan time.Time
	}
}

func NewGame() (*Game, error) {
	ebiten.SetWindowSize(GameWidth, GameHeight)
	ebiten.SetWindowTitle("Break'em")

	rand.Seed(time.Now().UnixNano())

	game := &Game{
		Lives: 5,
		Score: 0,
		Paddle: Paddle{
			Rect: image.Rectangle{
				Min: image.Point{WallSize, GameHeight - PaddleHeight},
				Max: image.Point{WallSize + PaddleWidth, GameHeight},
			},
		},
		Ball:   NewBall(),
		Bricks: make([]Brick, 0, 128),
	}

	game.Audio.Context = audio.NewContext(int(AudioSampleRate))

	{
		pos := image.Rectangle{
			image.Point{WallSize, WallSize * 4},
			image.Point{WallSize + BrickWidth, (WallSize * 4) + BrickHeight},
		}

		for i := 0; i < 9; i += 1 {
			game.Bricks = append(
				game.Bricks,
				Brick{
					Rect:   pos,
					Points: uint8(i),
				},
			)

			pos = pos.Add(image.Point{BrickWidth, 0})
		}
	}

	return game, nil
}

func NewBall() Ball {
	return Ball{
		Rect: image.Rectangle{
			Min: image.Point{WallSize, WallSize},
			Max: image.Point{WallSize + BallWidth, WallSize + BallHeight},
		},
		Vel: image.Point{
			rand.Intn(BallMaxSpeed/2) + 1,
			rand.Intn(BallMaxSpeed/2) + 1,
		},
	}
}

func (g *Game) Update() error {
	if g.Lives == 0 || len(g.Bricks) == 0 {
		return nil
	}

	if g.Audio.Stopper != nil {
		select {
		case <-g.Audio.Stopper:
			g.Audio.Tone.Stop()
			g.Audio.Stopper = nil
		default:
			break
		}
	}

	if g.Audio.NextTone != 0 {
		stopper := g.Audio.Tone.Play(g.Audio.NextTone)

		if stopper != nil {
			g.Audio.Stopper = stopper
		}

		g.Audio.NextTone = 0
	}

	// Allow for mouse controls, currently only when button held
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		cursorX, _ := ebiten.CursorPosition()
		diff := cursorX - (g.Paddle.Rect.Min.X + g.Paddle.Rect.Dx()/2)

		g.Paddle.Rect = g.Paddle.Rect.Add(image.Point{diff, 0})
		g.Paddle.Vel = diff
	} else {
		// Keyboard paddle movement, otherwise deceleration
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			g.Paddle.Vel -= PaddleSpeed
		} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
			g.Paddle.Vel += PaddleSpeed
		} else {
			if g.Paddle.Vel > 0 {
				g.Paddle.Vel -= PaddleDecel
			} else if g.Paddle.Vel < 0 {
				g.Paddle.Vel += PaddleDecel
			}
		}

		g.Paddle.Vel = Clamp(g.Paddle.Vel, -PaddleMaxSpeed, PaddleMaxSpeed)
		g.Paddle.Rect = g.Paddle.Rect.Add(image.Point{g.Paddle.Vel, 0})
	}

	// Clamp paddle to right wall boundary
	if diff := g.Paddle.Rect.Max.X - RightWall; diff > 0 {
		g.Paddle.Rect = g.Paddle.Rect.Sub(image.Point{diff, 0})
		g.Paddle.Vel = 0
	}

	// Clamp paddle to left wall boundary
	if diff := LeftWall - g.Paddle.Rect.Min.X; diff > 0 {
		g.Paddle.Rect = g.Paddle.Rect.Add(image.Point{diff, 0})
		g.Paddle.Vel = 0
	}

	g.Ball.Rect = g.Ball.Rect.Add(g.Ball.Vel)

	if g.Ball.Rect.Min.Y > GameHeight {
		g.Lives -= 1
		g.Ball = NewBall()
		return nil
	}

	// Clamp ball to right wall boundary
	if diff := g.Ball.Rect.Max.X - RightWall; diff > 0 {
		g.Ball.Rect = g.Ball.Rect.Sub(image.Point{diff, 0})
		g.Ball.Vel.X *= -1

		g.Audio.NextTone = rand.Intn(2)
	}

	// Clamp ball to top wall boundary
	if diff := LeftWall - g.Ball.Rect.Min.X; diff > 0 {
		g.Ball.Rect = g.Ball.Rect.Add(image.Point{diff, 0})
		g.Ball.Vel.X *= -1

		g.Audio.NextTone = rand.Intn(1) + 1
	}

	// Clamp ball to top wall boundary
	if diff := TopWall - g.Ball.Rect.Min.Y; diff > 0 {
		g.Ball.Rect = g.Ball.Rect.Add(image.Point{0, diff})
		g.Ball.Vel.Y *= -1

		g.Audio.NextTone = rand.Intn(1) + 1
	}

	// Ball intersection with paddle
	if diff := g.Ball.Rect.Intersect(g.Paddle.Rect); !diff.Empty() {
		size := diff.Size()

		g.Audio.NextTone = rand.Intn(2) + 1

		// Colliding on top
		if size.X > size.Y {
			if g.Paddle.Vel != 0 {
				g.Ball.Vel.X = g.Paddle.Vel
			} else {
				g.Ball.Rect = g.Ball.Rect.Sub(image.Point{0, size.Y})

				// Split the paddle into 1/4, 2/4, 1/4 sections, where the 1/4
				// section collisions cause the ball to move faster than the
				// center 2/4 collision
				divOne := g.Paddle.Rect.Min.X + g.Paddle.Rect.Dx()/4
				divTwo := g.Paddle.Rect.Max.X - g.Paddle.Rect.Dx()/4

				diffCenter := diff.Min.X + size.X/2

				// Paddle collision does not take bottom of paddle into account,
				// only top/sides are needed
				if diffCenter < divOne {
					g.Ball.Vel.X = -BallMaxSpeed / 2
					g.Ball.Vel.Y = -BallMaxSpeed / 3
				} else if diffCenter > divTwo {
					g.Ball.Vel.X = BallMaxSpeed / 2
					g.Ball.Vel.Y = -BallMaxSpeed / 3
				} else {
					g.Ball.Vel.Y = -BallMaxSpeed / 2
				}
			}
		} else {
			if diff.Min.X == g.Paddle.Rect.Min.X {
				// Left side of paddle
				g.Ball.Rect = g.Ball.Rect.Sub(image.Point{size.X, 0})
				g.Ball.Vel.X = -BallMaxSpeed
			} else {
				// Right side of paddle
				g.Ball.Rect = g.Ball.Rect.Add(image.Point{size.X, 0})
				g.Ball.Vel.X = BallMaxSpeed
			}

			g.Ball.Vel.Y *= -1
		}
	}

	remainingBricks := make([]Brick, 0, len(g.Bricks))

	brickCollide := false

	// Ball intersection with bricks
	for _, brick := range g.Bricks {
		diff := g.Ball.Rect.Intersect(brick.Rect)

		if diff.Empty() {
			remainingBricks = append(remainingBricks, brick)
			continue
		}

		brickCollide = true

		g.Score += brick.Points

		size := diff.Size()

		// Colliding on top/bottom
		if size.X > size.Y {
			g.Ball.Rect = g.Ball.Rect.Sub(image.Point{0, size.Y})

			// Where the paddle is split into 1/4, 2/4, 1/4, the bricks are
			// split into equal thirds
			divOne := brick.Rect.Min.X + brick.Rect.Dx()/3
			divTwo := brick.Rect.Max.X - brick.Rect.Dx()/3

			diffXCenter := diff.Min.X + (size.X / 2)
			diffYCenter := diff.Min.Y + (size.Y / 2)

			if diffXCenter < divOne {
				g.Ball.Vel.X = -BallMaxSpeed / 3
			} else if diffXCenter > divTwo {
				g.Ball.Vel.X = BallMaxSpeed / 3
			} else {
				// Ball can collide with either top or bottom of brick
				if diffYCenter < brick.Rect.Min.Y+brick.Rect.Dy()/2 {
					g.Ball.Vel.Y = -BallMaxSpeed / 2
				} else {
					g.Ball.Vel.Y = BallMaxSpeed / 2
				}
			}
		} else {
			if diff.Min.X == brick.Rect.Min.X {
				// Left side of brick
				g.Ball.Rect = g.Ball.Rect.Sub(image.Point{size.X, 0})
				g.Ball.Vel.X = -BallMaxSpeed / 2
			} else {
				// Right side of brick
				g.Ball.Rect = g.Ball.Rect.Add(image.Point{size.X, 0})
				g.Ball.Vel.X = BallMaxSpeed / 2
			}

			g.Ball.Vel.Y *= -1
		}
	}

	if brickCollide {
		g.Audio.NextTone = rand.Intn(3) + 1

		g.Bricks = remainingBricks
	}

	g.Ball.Vel.X = Clamp(g.Ball.Vel.X, -BallMaxSpeed, BallMaxSpeed)
	g.Ball.Vel.Y = Clamp(g.Ball.Vel.Y, -BallMaxSpeed, BallMaxSpeed)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if g.Lives == 0 {
		ebitenutil.DebugPrint(screen, "Game Over")
		return
	}

	if len(g.Bricks) == 0 {
		ebitenutil.DebugPrint(screen, "You Win!")
		return
	}

	width, height := screen.Size()

	min := image.Point{}
	max := image.Point{width, height}

	wallColor := color.RGBA{128, 128, 128, 255}

	// Top Wall
	{
		rect := image.Rectangle{min, max}
		rect.Max.Y = TopWall

		sub := screen.SubImage(rect).(*ebiten.Image)
		sub.Fill(wallColor)
	}

	// Left Wall
	{
		rect := image.Rectangle{min, max}
		rect.Max.X /= 20

		sub := screen.SubImage(rect).(*ebiten.Image)
		sub.Fill(wallColor)
	}

	// Right Wall
	{
		rect := image.Rectangle{min, max}
		rect.Min.X = width - (width / 20)

		sub := screen.SubImage(rect).(*ebiten.Image)
		sub.Fill(wallColor)
	}

	// Paddle
	{
		sub := screen.SubImage(g.Paddle.Rect).(*ebiten.Image)
		sub.Fill(color.RGBA{255, 0, 0, 255})
	}

	// Ball
	{
		sub := screen.SubImage(g.Ball.Rect).(*ebiten.Image)
		sub.Fill(color.RGBA{255, 255, 255, 255})
	}

	// Bricks
	for _, brick := range g.Bricks {
		sub := screen.SubImage(brick.Rect).(*ebiten.Image)
		sub.Fill(color.RGBA{255, 255, 0, 255})
	}

	// Score, Lives, etc.
	{
		ebitenutil.DebugPrint(screen, strconv.Itoa(int(g.Score)))
		ebitenutil.DebugPrintAt(screen, strconv.Itoa(g.Lives), GameWidth-20, 0)
	}
}

func (g *Game) Layout(int, int) (int, int) {
	return GameWidth, GameHeight
}
