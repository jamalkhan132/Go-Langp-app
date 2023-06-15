package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/contentstream"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/model"
)

func main() {
	// Set the license key (optional).
	license.SetLicenseKey("YOUR_LICENSE_KEY", "")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			http.ServeFile(w, r, "index.html")
			return
		}

		err := r.ParseMultipartForm(10 << 20) // Limit file size to 10MB
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		file, handler, err := r.FormFile("pdf")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		fmt.Printf("Received file: %s\n", handler.Filename)

		// Read the uploaded PDF file.
		pdfData, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Load the PDF document.
		pdfReader, err := model.NewPdfReader(bytes.NewReader(pdfData))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create a new Excel file.
		excelFile := excelize.NewFile()
		sheetName := "Sheet1"

		// Add a blue color style to be applied to the rows.
		styleID, err := excelFile.NewStyle(`{"fill":{"type":"pattern","color":["#1f497d"],"pattern":1}}`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Iterate through each page of the PDF.
		numPages, err := pdfReader.GetNumPages()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for pageNum := 1; pageNum <= numPages; pageNum++ {
			page, err := pdfReader.GetPage(pageNum)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Extract the text content from the page.
			contentStreams, err := page.GetContentStreams()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Iterate through each content stream on the page.
			for _, contentStream := range contentStreams {
				parser := contentstream.NewContentStreamParser(contentStream)
				operations, err := parser.Parse()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Process each text element in the content stream.
				for _, op := range *operations {
					if op.Operand == "Tj" {
						// Text element found.
						if len(op.Params) == 1 {
							// Extract the text value.
							text, ok := op.Params[0].(*core.PdfObjectString)
							if ok {
								// Add the text value to the Excel sheet.
								cell := fmt.Sprintf("A%d", pageNum)
								excelFile.SetCellValue(sheetName, cell, text.Str())

								// Add the blue color style to the row.
								row := fmt.Sprintf("%s%d:%s%d", "A", pageNum, "A", pageNum)
								excelFile.SetCellStyle(sheetName, row, row, styleID)
							}
						}
					}
				}
			}
		}

		// Save the Excel file to a temporary location.
		tempFilePath := "./result.xlsx"

		// Create the directory if it doesn't exist.
		err = os.MkdirAll(filepath.Dir(tempFilePath), os.ModePerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save the Excel file.
		err = excelFile.SaveAs(tempFilePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Provide a download link to the generated Excel file.
		downloadLink := fmt.Sprintf(`<a href="/download?path=%s" download>Download Excel</a>`, tempFilePath)

		responseHTML := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>PDF to Excel Conversion</title>
				<style>
					.bar {
						background-color: blue;
						height: 10px;
					}
				</style>
			</head>
			<body>
				<div class="bar"></div>
				<h1>PDF to Excel Conversion</h1>
				<p>Conversion completed successfully.</p>
				<p>%s</p>
			</body>
			</html>
		`, downloadLink)

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(responseHTML))
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			http.Error(w, "Invalid file path", http.StatusBadRequest)
			return
		}

		http.ServeFile(w, r, filePath)
	})

	log.Fatal(http.ListenAndServe(":8092", nil))
}
