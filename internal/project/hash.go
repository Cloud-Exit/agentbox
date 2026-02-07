// ExitBox - Multi-Agent Container Sandbox
// Copyright (C) 2026 Cloud Exit B.V.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package project

// POSIXCksum computes the POSIX cksum CRC32 for a byte slice.
// This is NOT the same as Go's crc32.IEEE — POSIX cksum uses a different
// polynomial bit ordering and appends the byte count to the CRC.
func POSIXCksum(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = (crc << 8) ^ crcTable[((crc>>24)^uint32(b))&0xFF]
	}
	// Fold in the length
	length := len(data)
	for length > 0 {
		crc = (crc << 8) ^ crcTable[((crc>>24)^uint32(length&0xFF))&0xFF]
		length >>= 8
	}
	return ^crc & 0xFFFFFFFF
}

// POSIXCksumString computes the POSIX cksum CRC32 for a string.
func POSIXCksumString(s string) uint32 {
	return POSIXCksum([]byte(s))
}

// crcTable is the pre-computed lookup table for the POSIX CRC polynomial.
// This matches the output of the POSIX cksum command.
var crcTable [256]uint32

func init() {
	const poly uint32 = 0x04C11DB7
	for i := 0; i < 256; i++ {
		crc := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		crcTable[i] = crc
	}
}
