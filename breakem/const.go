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

const (
	GameWidth  = 500
	GameHeight = 500

	WallSize = 25

	LeftWall  = WallSize
	RightWall = GameWidth - WallSize
	TopWall   = WallSize

	BrickWidth  = WallSize * 2
	BrickHeight = WallSize

	PaddleWidth    = WallSize * 4
	PaddleHeight   = WallSize / 2
	PaddleSpeed    = 5
	PaddleMaxSpeed = 10
	PaddleDecel    = 2

	BallWidth    = 5
	BallHeight   = BallWidth
	BallMaxSpeed = 10
)
