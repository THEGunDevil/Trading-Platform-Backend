package handlers

// import (
// 	"encoding/csv"
// 	"fmt"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
// 	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
// 	"github.com/gin-gonic/gin"
// 	"github.com/jackc/pgx/v5/pgtype"
// 	"github.com/phpdave11/gofpdf"
// 	"github.com/xuri/excelize/v2"
// )

// // --- PDF Helpers ---
// func setupPDF(title string) *gofpdf.Fpdf {
// 	pdf := gofpdf.New("L", "mm", "A4", "")
// 	pdf.SetMargins(15, 15, 15)
// 	pdf.AddPage()
// 	pdf.SetFont("Helvetica", "B", 16)
// 	pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
// 	pdf.Ln(5)
// 	pdf.SetFont("Helvetica", "", 10)
// 	pdf.CellFormat(0, 6, fmt.Sprintf("Generated on %s", time.Now().Format("2006-01-02 15:04:05")), "", 1, "C", false, 0, "")
// 	pdf.Ln(5)
// 	pdf.SetFooterFunc(func() {
// 		pdf.SetY(-15)
// 		pdf.SetFont("Helvetica", "I", 8)
// 		pdf.SetTextColor(128, 128, 128)
// 		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")
// 	})
// 	return pdf
// }

// // --- XLSX Writer ---
// func writeXLSX(c *gin.Context, f *excelize.File, filename string) {
// 	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
// 	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
// 	if err := f.Write(c.Writer); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	}
// }

// // --- Helpers for PDF Table ---
// func getDynamicWidths(headers []string, rows [][]string, minWidth float64, maxWidth float64) []float64 {
// 	widths := make([]float64, len(headers))
// 	for i := range headers {
// 		width := float64(len(headers[i])*2 + 10) // header width
// 		for _, row := range rows {
// 			if i < len(row) {
// 				cellLen := float64(len(row[i])*2 + 10)
// 				if cellLen > width {
// 					width = cellLen
// 				}
// 			}
// 		}
// 		if width < minWidth {
// 			width = minWidth
// 		} else if width > maxWidth {
// 			width = maxWidth
// 		}
// 		widths[i] = width
// 	}
// 	return widths
// }

// func drawTableHeader(pdf *gofpdf.Fpdf, headers []string, widths []float64) {
// 	pdf.SetFont("Helvetica", "B", 10)
// 	pdf.SetFillColor(180, 180, 255)
// 	pdf.SetTextColor(0, 0, 0)
// 	pdf.SetDrawColor(100, 100, 100)
// 	for i, header := range headers {
// 		pdf.CellFormat(widths[i], 9, header, "1", 0, "C", true, 0, "")
// 	}
// 	pdf.Ln(-1)
// }

// func drawTableRow(pdf *gofpdf.Fpdf, row []string, widths []float64, rowIndex int) {
// 	pdf.SetFont("Helvetica", "", 9)
// 	if rowIndex%2 == 0 {
// 		pdf.SetFillColor(245, 245, 245)
// 	} else {
// 		pdf.SetFillColor(255, 255, 255)
// 	}
// 	pdf.SetTextColor(0, 0, 0)
// 	pdf.SetDrawColor(180, 180, 180)
// 	for i, cell := range row {
// 		pdf.CellFormat(widths[i], 8, cell, "1", 0, "L", true, 0, "")
// 	}
// 	pdf.Ln(-1)
// }

// // parsePagination reads ?page= and ?limit= query parameters, returns defaults if missing
// func parsePagination(c *gin.Context) (page, limit int) {
// 	page = 1
// 	limit = 10

// 	if p := c.Query("page"); p != "" {
// 		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
// 			page = parsed
// 		}
// 	}

// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
// 			limit = parsed
// 		}
// 	}

// 	return
// }

// // --- Books Download ---
// func DownloadSearchBooksHandler(c *gin.Context) {
// 	format := c.Query("format")
// 	genre := c.Query("genre")   // e.g., "fiction" or "all"
// 	// search := c.Query("search") // search term
// 	page, limit := parsePagination(c)
// 	offset := (page - 1) * limit

// 	params := gen.SearchBooksWithPaginationParams{
// 		Column1: genre,
// 		// Column2: pgtype.Text{String: search, Valid: true},
// 		Limit:   int32(limit),
// 		Offset:  int32(offset),
// 	}

// 	books, err := db.Q.SearchBooksWithPagination(c.Request.Context(), params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	switch format {
// 	case "csv":
// 		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=search_books_page_%d.csv", page))
// 		c.Header("Content-Type", "text/csv")
// 		writer := csv.NewWriter(c.Writer)
// 		defer writer.Flush()
// 		writer.Write([]string{"ID", "Title", "Author", "Genre", "Published Year", "ISBN", "Available Copies", "Total Copies"})
// 		for _, book := range books {
// 			writer.Write([]string{
// 				book.ID.String(),
// 				book.Title,
// 				book.Author,
// 				book.Genre,
// 				fmt.Sprintf("%d", book.PublishedYear.Int32),
// 				book.Isbn.String,
// 				fmt.Sprintf("%d", book.AvailableCopies.Int32),
// 				fmt.Sprintf("%d", book.TotalCopies),
// 			})
// 		}

// 	case "pdf":
// 		pdf := setupPDF(fmt.Sprintf("Books Search Report - Page %d", page))
// 		rows := [][]string{}
// 		for _, book := range books {
// 			rows = append(rows, []string{
// 				book.ID.String(),
// 				book.Title,
// 				book.Author,
// 				book.Genre,
// 				fmt.Sprintf("%d", book.PublishedYear.Int32),
// 				book.Isbn.String,
// 				fmt.Sprintf("%d", book.AvailableCopies.Int32),
// 				fmt.Sprintf("%d", book.TotalCopies),
// 			})
// 		}
// 		headers := []string{"ID", "Title", "Author", "Genre", "Published Year", "ISBN", "Available Copies", "Total Copies"}
// 		widths := getDynamicWidths(headers, rows, 20, 80)
// 		drawTableHeader(pdf, headers, widths)
// 		for i, row := range rows {
// 			drawTableRow(pdf, row, widths, i)
// 		}
// 		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=search_books_page_%d.pdf", page))
// 		c.Header("Content-Type", "application/pdf")
// 		pdf.Output(c.Writer)

// 	case "xlsx":
// 		f := excelize.NewFile()
// 		sheet := "SearchBooks"
// 		f.NewSheet(sheet)
// 		f.DeleteSheet("Sheet1")
// 		headers := []string{"ID", "Title", "Author", "Genre", "Published Year", "ISBN", "Available Copies", "Total Copies"}
// 		for i, h := range headers {
// 			col := string(rune('A' + i))
// 			f.SetCellValue(sheet, fmt.Sprintf("%s1", col), h)
// 		}
// 		for r, book := range books {
// 			values := []interface{}{
// 				book.ID.String(),
// 				book.Title,
// 				book.Author,
// 				book.Genre,
// 				book.PublishedYear.Int32,
// 				book.Isbn.String,
// 				book.AvailableCopies.Int32,
// 				book.TotalCopies,
// 			}
// 			for i, v := range values {
// 				col := string(rune('A' + i))
// 				f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, r+2), v)
// 			}
// 		}
// 		writeXLSX(c, f, fmt.Sprintf("search_books_page_%d.xlsx", page))

// 	default:
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format, use ?format=csv, ?format=pdf or ?format=xlsx"})
// 	}
// }

// // --- Borrows Download ---
// func DownloadBorrowsHandler(c *gin.Context) {
// 	format := c.Query("format")

// 	// Get search column and value from query params
// 	column := c.Query("column") // "user_name" or "book_title"
// 	search := c.Query("search") // the text to search

// 	// Parse limit and offset
// 	limit := int32(10)
// 	offset := int32(0)
// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
// 			limit = int32(parsed)
// 		}
// 	}
// 	if o := c.Query("offset"); o != "" {
// 		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
// 			offset = int32(parsed)
// 		}
// 	}

// 	params := gen.SearchBorrowsWithPaginationParams{
// 		Column1: column,
// 		Column2: pgtype.Text{String: search, Valid: true},
// 		Limit:   limit,
// 		Offset:  offset,
// 	}

// 	borrows, err := db.Q.SearchBorrowsWithPagination(c.Request.Context(), params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// --- CSV / PDF / XLSX handlers ---
// 	switch format {
// 	case "csv":
// 		c.Header("Content-Disposition", "attachment; filename=borrows.csv")
// 		c.Header("Content-Type", "text/csv")
// 		writer := csv.NewWriter(c.Writer)
// 		defer writer.Flush()
// 		writer.Write([]string{"Borrow ID", "User ID", "Book ID", "Borrowed At", "Due Date", "Returned At", "Book Title", "User Name"})
// 		for _, b := range borrows {
// 			returned := ""
// 			if b.ReturnedAt.Valid {
// 				returned = b.ReturnedAt.Time.Format("2006-01-02 15:04:05")
// 			}
// 			writer.Write([]string{
// 				b.ID.String(),
// 				b.UserID.String(),
// 				b.BookID.String(),
// 				b.BorrowedAt.Time.Format("2006-01-02 15:04:05"),
// 				b.DueDate.Time.Format("2006-01-02 15:04:05"),
// 				returned,
// 				b.BookTitle,
// 				b.UserName.(string)})
// 		}
// 	case "pdf":
// 		pdf := setupPDF("Borrows Report")
// 		headers := []string{"Borrow ID", "User ID", "Book ID", "Borrowed At", "Due Date", "Returned At", "Book Title", "User Name"}
// 		rows := [][]string{}
// 		for _, b := range borrows {
// 			returned := "Not Returned"
// 			if b.ReturnedAt.Valid {
// 				returned = b.ReturnedAt.Time.Format("2006-01-02 15:04:05")
// 			}
// 			rows = append(rows, []string{
// 				b.ID.String(),
// 				b.UserID.String(),
// 				b.BookID.String(),
// 				b.BorrowedAt.Time.Format("2006-01-02 15:04:05"),
// 				b.DueDate.Time.Format("2006-01-02 15:04:05"),
// 				returned,
// 				b.BookTitle,
// 				b.UserName.(string),
// 			})
// 		}
// 		widths := getDynamicWidths(headers, rows, 20, 80)
// 		drawTableHeader(pdf, headers, widths)
// 		for i, row := range rows {
// 			drawTableRow(pdf, row, widths, i)
// 		}
// 		c.Header("Content-Disposition", "attachment; filename=borrows.pdf")
// 		c.Header("Content-Type", "application/pdf")
// 		pdf.Output(c.Writer)
// 	case "xlsx":
// 		f := excelize.NewFile()
// 		sheet := "Borrows"
// 		f.NewSheet(sheet)
// 		f.DeleteSheet("Sheet1")
// 		headers := []string{"Borrow ID", "User ID", "Book ID", "Borrowed At", "Due Date", "Returned At", "Book Title", "User Name"}
// 		for i, h := range headers {
// 			col := string(rune('A' + i))
// 			f.SetCellValue(sheet, fmt.Sprintf("%s1", col), h)
// 		}
// 		for r, b := range borrows {
// 			returned := "Not Returned"
// 			if b.ReturnedAt.Valid {
// 				returned = b.ReturnedAt.Time.Format("2006-01-02 15:04:05")
// 			}
// 			values := []interface{}{
// 				b.ID.String(),
// 				b.UserID.String(),
// 				b.BookID.String(),
// 				b.BorrowedAt.Time.Format("2006-01-02 15:04:05"),
// 				b.DueDate.Time.Format("2006-01-02 15:04:05"),
// 				returned,
// 				b.BookTitle,
// 				b.UserName,
// 			}
// 			for i, v := range values {
// 				col := string(rune('A' + i))
// 				f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, r+2), v)
// 			}
// 		}
// 		writeXLSX(c, f, "borrows.xlsx")
// 	default:
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
// 	}
// }

// // --- Users Download ---
// func DownloadUsersHandler(c *gin.Context) {
// 	format := c.Query("format")
// 	search := c.Query("search") // for email filtering

// 	// parse limit and offset
// 	limit := 10
// 	offset := 0
// 	if l := c.Query("limit"); l != "" {
// 		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
// 			limit = parsed
// 		}
// 	}
// 	if o := c.Query("offset"); o != "" {
// 		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
// 			offset = parsed
// 		}
// 	}

// 	params := gen.SearchUsersByEmailWithPaginationParams{
// 		Column1: search, // search query
// 		Limit:   int32(limit),
// 		Offset:  int32(offset),
// 	}

// 	users, err := db.Q.SearchUsersByEmailWithPagination(c.Request.Context(), params)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	switch format {
// 	case "csv":
// 		c.Header("Content-Disposition", "attachment; filename=users.csv")
// 		c.Header("Content-Type", "text/csv")
// 		writer := csv.NewWriter(c.Writer)
// 		defer writer.Flush()
// 		writer.Write([]string{"ID", "First Name", "Last Name", "Email", "Role", "Created At"})
// 		for _, u := range users {
// 			writer.Write([]string{
// 				u.ID.String(),
// 				u.FirstName,
// 				u.LastName,
// 				u.Email,
// 				u.Role.String,
// 				u.CreatedAt.Time.Format("2006-01-02 15:04:05"),
// 			})
// 		}
// 	case "pdf":
// 		pdf := setupPDF("Users Report")
// 		headers := []string{"ID", "First Name", "Last Name", "Email", "Role", "Created At"}
// 		rows := [][]string{}
// 		for _, u := range users {
// 			rows = append(rows, []string{
// 				u.ID.String(),
// 				u.FirstName,
// 				u.LastName,
// 				u.Email,
// 				u.Role.String,
// 				u.CreatedAt.Time.Format("2006-01-02 15:04:05"),
// 			})
// 		}
// 		widths := getDynamicWidths(headers, rows, 20, 60)
// 		drawTableHeader(pdf, headers, widths)
// 		for i, row := range rows {
// 			drawTableRow(pdf, row, widths, i)
// 		}
// 		c.Header("Content-Disposition", "attachment; filename=users.pdf")
// 		c.Header("Content-Type", "application/pdf")
// 		pdf.Output(c.Writer)
// 	case "xlsx":
// 		f := excelize.NewFile()
// 		sheet := "Users"
// 		f.NewSheet(sheet)
// 		f.DeleteSheet("Sheet1")
// 		headers := []string{"ID", "First Name", "Last Name", "Email", "Role", "Created At"}
// 		for i, h := range headers {
// 			col := string(rune('A' + i))
// 			f.SetCellValue(sheet, fmt.Sprintf("%s1", col), h)
// 		}
// 		for r, u := range users {
// 			values := []interface{}{
// 				u.ID.String(),
// 				u.FirstName,
// 				u.LastName,
// 				u.Email,
// 				u.Role.String,
// 				u.CreatedAt.Time.Format("2006-01-02 15:04:05"),
// 			}
// 			for i, v := range values {
// 				col := string(rune('A' + i))
// 				f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, r+2), v)
// 			}
// 		}
// 		writeXLSX(c, f, "users.xlsx")
// 	default:
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
// 	}
// }
