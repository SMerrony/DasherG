// dasherg.go

// Copyright ©2021,2025 Steve Merrony

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Special Key buttons...
type buttonTheme struct{}

var _ fyne.Theme = (*buttonTheme)(nil)

func (t *buttonTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameForeground {
		return color.White
	}
	if name == theme.ColorNameButton {
		return color.RGBA{0, 200, 225, 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *buttonTheme) Font(textStyle fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(textStyle)
}

func (t *buttonTheme) Icon(themeIconName fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(themeIconName)
}

func (t *buttonTheme) Size(themeSize fyne.ThemeSizeName) (f float32) {
	if themeSize == theme.SizeNameInnerPadding {
		return 1.0
	}
	if themeSize == theme.SizeNameText {
		return 12.0
	}
	return theme.DefaultTheme().Size(themeSize)
}

// Function Key buttons...
type fkeyTheme struct{}

var _ fyne.Theme = (*fkeyTheme)(nil)

func (t *fkeyTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameForeground {
		return color.White
	}
	if name == theme.ColorNameButton {
		return color.RGBA{0, 200, 225, 255}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *fkeyTheme) Font(textStyle fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(textStyle)
}

func (t *fkeyTheme) Icon(themeIconName fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(themeIconName)
}

func (t *fkeyTheme) Size(themeSize fyne.ThemeSizeName) (f float32) {
	return theme.DefaultTheme().Size(themeSize)
}

// Function Key Template Labels...
type fkeyLabelTheme struct{}

var _ fyne.Theme = (*fkeyLabelTheme)(nil)

func (t *fkeyLabelTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground { // TODO - not working
		return color.RGBA{255, 253, 208, 255}
	}
	if name == theme.ColorNameSeparator {
		return color.Black
	}
	if name == theme.ColorNameForeground {
		return color.Black
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *fkeyLabelTheme) Font(textStyle fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(textStyle)
}

func (t *fkeyLabelTheme) Icon(themeIconName fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(themeIconName)
}

func (t *fkeyLabelTheme) Size(themeSize fyne.ThemeSizeName) (f float32) {
	if themeSize == theme.SizeNameInnerPadding {
		return 1.0
	}
	if themeSize == theme.SizeNameText {
		return 9.0
	}
	return theme.DefaultTheme().Size(themeSize)
}
