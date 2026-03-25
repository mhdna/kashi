package main

import (
	"image/jpeg"
	"log"
	"os"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/ean"
	"github.com/signintech/gopdf"
)

func generatePDF() {
	// Create the barcode
	c, err := ean.Encode("8888880005436")
	if err != nil {
		panic(err)
	}

	eanImg, _ := barcode.Scale(c, 200, 50)
	// encode the barcode as png
	// png.Encode(file, img)
	file, err := os.Create("./img.jpeg")
	if err != nil {
		panic(err)
	}
	jpeg.Encode(file, eanImg, &jpeg.Options{100})

	pdf := gopdf.GoPdf{}

	pdf.Start(gopdf.Config{PageSize: gopdf.Rect{W: 164, H: 85}})
	pdf.AddPage()

	pdf.Image("./img.jpeg", 5, 10, nil)

	err = pdf.AddTTFFont("noto", "./Georgia.ttf")
	if err != nil {
		log.Print(err.Error())
		return
	}

	err = pdf.SetFont("noto", "", 12)
	if err != nil {
		log.Print(err.Error())
		return
	}
	pdf.SetXY(40, 70)
	pdf.Cell(nil, "Kids Boy Sweatshirt")
	pdf.SetXY(40, 80)
	pdf.Cell(nil, "8888880005436")
	pdf.WritePdf("test.pdf")
}
