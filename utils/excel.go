package utils

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type Worksheet struct {
	Name string
	Rows [][]string
}

// WriteSimpleXLSX writes a minimal XLSX with the provided worksheets.
func WriteSimpleXLSX(filename string, sheets []Worksheet) error {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// [Content_Types].xml
	content := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
    <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
    <Default Extension="xml" ContentType="application/xml"/>
    <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
%s
</Types>`
	var overrides strings.Builder
	for i := range sheets {
		overrides.WriteString(fmt.Sprintf(`    <Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`+"\n", i+1))
	}
	if err := addZipFile(zw, "[Content_Types].xml", fmt.Sprintf(content, overrides.String())); err != nil {
		return err
	}

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
    <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	if err := addZipFile(zw, "_rels/.rels", rels); err != nil {
		return err
	}

	// xl/_rels/workbook.xml.rels
	var wbRels strings.Builder
	wbRels.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	wbRels.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + "\n")
	for i := range sheets {
		wbRels.WriteString(fmt.Sprintf(`    <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`+"\n", i+1, i+1))
	}
	wbRels.WriteString(`</Relationships>`)
	if err := addZipFile(zw, "xl/_rels/workbook.xml.rels", wbRels.String()); err != nil {
		return err
	}

	// xl/workbook.xml
	var wb strings.Builder
	wb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	wb.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` + "\n")
	wb.WriteString(`<sheets>` + "\n")
	for i, sh := range sheets {
		wb.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`+"\n", xmlEscape(sh.Name), i+1, i+1))
	}
	wb.WriteString(`</sheets></workbook>`)
	if err := addZipFile(zw, "xl/workbook.xml", wb.String()); err != nil {
		return err
	}

	// worksheets
	for i, sh := range sheets {
		xmlStr := buildSheetXML(sh.Rows)
		if err := addZipFile(zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), xmlStr); err != nil {
			return err
		}
	}

	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(filename, buf.Bytes(), 0644)
}

func addZipFile(zw *zip.Writer, name string, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(content))
	return err
}

func buildSheetXML(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetData>`)
	for rIdx, row := range rows {
		b.WriteString(fmt.Sprintf(`<row r="%d">`, rIdx+1))
		for cIdx, cell := range row {
			col := columnName(cIdx + 1)
			b.WriteString(fmt.Sprintf(`<c r="%s%d" t="inlineStr"><is><t>%s</t></is></c>`, col, rIdx+1, xmlEscape(cell)))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func columnName(n int) string {
	result := ""
	for n > 0 {
		n--
		result = string(rune('A'+(n%26))) + result
		n /= 26
	}
	return result
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
