// makeicon 生成 icon.ico：绿色方块 + 左上角黑色 ">."（用 Cubic_11 字体渲染）
// 用法: go run ./tools/makeicon
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// drawIcon 渲染 size×size 图标：绿色底，左上角黑色 ">."
// 像素字体 1:1 渲染：小尺寸用整数 hinting 字号直接绘制（无膨胀），
// 大尺寸按比例放大字号并适度膨胀，避免下采样破坏笔画
func drawIcon(size int, f *sfnt.Font) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	green := color.RGBA{0, 255, 102, 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, green)
		}
	}
	dilate1 := func() {
		orig := image.NewRGBA(img.Bounds())
		copy(orig.Pix, img.Pix)
		isBlack := func(x, y int) bool {
			if x < 0 || y < 0 || x >= size || y >= size {
				return false
			}
			c := orig.RGBAAt(x, y)
			return c.R == 0 && c.G == 0 && c.B == 0
		}
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if isBlack(x, y) {
					continue
				}
				hit := false
				for dy := -1; dy <= 1 && !hit; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if isBlack(x+dx, y+dy) {
							hit = true
							break
						}
					}
				}
				if hit {
					img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
				}
			}
		}
	}
	// 字号:小尺寸取画布能容纳的最大整数档(">." 宽 = 2×字号),
	// 大尺寸按比例。膨胀仅大尺寸需要(像素字体笔画恒定 1px)。
	fontSize, margin := float64(size-2)/2, 1
	dilations := 0
	if size > 32 {
		fontSize = float64(size) * 0.43
		margin = size / 16
		dilations = size / 64 // 48:0, 64:1, 128:2, 256:4
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	defer face.Close()
	ascent := face.Metrics().Ascent.Ceil()
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
		Dot:  fixed.P(margin, margin+ascent),
	}
	d.DrawString(">.")
	// 二值化:抗锯齿灰度全部转纯黑,再做像素膨胀加粗
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := img.RGBAAt(x, y)
			if !(c.G > 200 && c.B > 50 && c.R < 40) { // 非纯绿背景
				img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	for i := 0; i < dilations; i++ {
		dilate1()
	}
	return img
}

// dib 输出 32bpp DIB 数据（自下而上 BGRA + AND mask），供 ICO 使用
func dib(img *image.RGBA) []byte {
	s := img.Bounds().Dx()
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(40))        // BITMAPINFOHEADER
	binary.Write(&b, binary.LittleEndian, int32(s))          // width
	binary.Write(&b, binary.LittleEndian, int32(s*2))        // height 含 AND mask
	binary.Write(&b, binary.LittleEndian, uint16(1))         // planes
	binary.Write(&b, binary.LittleEndian, uint16(32))        // bpp
	binary.Write(&b, binary.LittleEndian, uint32(0))         // BI_RGB
	binary.Write(&b, binary.LittleEndian, uint32(s*s*4))     // 图像字节数
	binary.Write(&b, binary.LittleEndian, int32(0))          // xppm
	binary.Write(&b, binary.LittleEndian, int32(0))          // yppm
	binary.Write(&b, binary.LittleEndian, uint32(0))         // clrUsed
	binary.Write(&b, binary.LittleEndian, uint32(0))         // clrImportant
	for y := s - 1; y >= 0; y-- {                            // 自下而上
		for x := 0; x < s; x++ {
			c := img.RGBAAt(x, y)
			b.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	maskRow := (s + 31) / 32 * 4 // 1bpp AND mask 每行 4 字节对齐
	for y := 0; y < s; y++ {
		b.Write(make([]byte, maskRow))
	}
	return b.Bytes()
}

func encodeICO(sizes []int, f *sfnt.Font) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))
	offset := 6 + 16*len(sizes)
	datas := make([][]byte, len(sizes))
	for i, s := range sizes {
		datas[i] = dib(drawIcon(s, f))
		w, h := byte(s), byte(s)
		if s >= 256 {
			w, h = 0, 0
		}
		binary.Write(&buf, binary.LittleEndian, w)
		binary.Write(&buf, binary.LittleEndian, h)
		binary.Write(&buf, binary.LittleEndian, uint8(0))     // palette
		binary.Write(&buf, binary.LittleEndian, uint8(0))     // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))    // planes
		binary.Write(&buf, binary.LittleEndian, uint16(32))   // bpp
		binary.Write(&buf, binary.LittleEndian, uint32(len(datas[i])))
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(datas[i])
	}
	for _, d := range datas {
		buf.Write(d)
	}
	return buf.Bytes()
}

func main() {
	const ttfPath = "tools/makeicon/Cubic_11.ttf"
	fontBytes, err := os.ReadFile(ttfPath)
	if err != nil {
		panic("缺少 " + ttfPath + "，请先转换：\n  python -m fontTools.ttLib.woff2 decompress Cubic_11.woff2 -o " + ttfPath)
	}
	f, err := sfnt.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	// Windows 各 DPI 缩放档位对应的标准图标尺寸
	if err := os.WriteFile("icon.ico", encodeICO([]int{16, 20, 24, 32, 40, 48, 64, 96, 128, 256}, f), 0o644); err != nil {
		panic(err)
	}
}
