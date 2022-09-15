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
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Tone struct {
	Player *audio.Player
}

func (tone *Tone) Play(note int) <-chan time.Time {
	if tone.Player != nil {
		return nil
	}

	var err error
	tone.Player, err = audio.CurrentContext().NewPlayer(
		&Stream{Frequency: 440.0 * float64(note)},
	)

	if err != nil {
		return nil
	}

	tone.Player.Play()

	return time.After(50 * time.Millisecond)
}

func (tone *Tone) Stop() {
	if tone.Player != nil {
		tone.Player.Close()
		tone.Player = nil
	}
}

type Stream struct {
	Position  int64
	Remainder []byte
	Frequency float64
}

func (stream *Stream) Read(buf []byte) (int, error) {
	if len(stream.Remainder) > 0 {
		n := copy(buf, stream.Remainder)
		stream.Remainder = stream.Remainder[n:]
		return n, nil
	}

	var bak []byte

	if len(buf)%4 > 0 {
		bak = buf
		buf = make([]byte, len(bak)+4-len(bak)%4)
	}

	length := int64(AudioSampleRate / stream.Frequency)

	p := stream.Position / 4
	for i := 0; i < len(buf)/4; i += 1 {
		fsample := math.Sin(2.0*math.Pi*float64(p)/float64(length))
		isample := int16(fsample * float64(math.MaxInt16))

		buf[4*i+0] = byte(isample)
		buf[4*i+1] = byte(isample >> 8)
		buf[4*i+2] = byte(isample)
		buf[4*i+3] = byte(isample >> 8)

		p += 1
	}

	stream.Position += int64(len(buf))
	stream.Position %= length * 4

	if bak != nil {
		n := copy(bak, buf)
		stream.Remainder = buf[n:]
		return n, nil
	}

	return len(buf), nil
}
