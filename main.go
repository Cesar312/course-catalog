package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const PORT = ":8000"

type CourseCatalog struct {
	CID     string
	CNAME   string
	CPREREQ string
}

var db *sql.DB

func main() {
	// Connect to database
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//Create table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS catalog (cid text PRIMARY KEY, cname text, cprereq text)")
	if err != nil {
		log.Fatal(err)
	}

	// Create router
	mux := http.NewServeMux()
	s := &http.Server{
		Addr:         PORT,
		Handler:      mux,
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	mux.HandleFunc("/", defaultHandler)
	mux.HandleFunc("/delete/", deleteHandler)
	mux.HandleFunc("/insert/", insertHandler)
	mux.HandleFunc("/list", listHandler)
	mux.HandleFunc("/search/", searchHandler)
	mux.HandleFunc("/status", statusHandler)

	fmt.Println("Ready to serve at", PORT)
	err = s.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

func insert(course *CourseCatalog) error {
	_, err := db.Exec("INSERT INTO catalog (cid, cname, cprereq) VALUES ($1, $2, $3)",
		course.CID, course.CNAME, course.CPREREQ)
	if err != nil {
		return fmt.Errorf("could not insert course: %v", err)
	}
	return nil
}

func deleteEntry(CID string) error {
	_, err := db.Exec("DELETE FROM catalog WHERE cid = $1", CID)
	if err != nil {
		return fmt.Errorf("could not delete course: %v", err)
	}
	return nil
}

func search(CID string) (*CourseCatalog, error) {
	course := &CourseCatalog{}
	err := db.QueryRow("SELECT cid, cname, cprereq FROM catalog WHERE cid = $1", CID).Scan(&course.CID, &course.CNAME, &course.CPREREQ)
	if err != nil {
		return nil, fmt.Errorf("could not find course: %v", err)
	}
	return course, nil
}

func list() ([]CourseCatalog, error) {
	rows, err := db.Query("SELECT cid, cname, cprereq FROM catalog")
	if err != nil {
		return nil, fmt.Errorf("could not list courses: %v", err)
	}
	defer rows.Close()

	var courses []CourseCatalog
	for rows.Next() {
		var c CourseCatalog
		if err := rows.Scan(&c.CID, &c.CNAME, &c.CPREREQ); err != nil {
			return nil, fmt.Errorf("could not scan course: %v", err)
		}
		courses = append(courses, c)
	}
	return courses, nil
}

func initCourse(CID, CNAME, CPREREQ string) *CourseCatalog {
	return &CourseCatalog{CID: CID, CNAME: CNAME, CPREREQ: CPREREQ}
}

// Handler functions
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving:", r.URL.Path, "from", r.Host)
	w.WriteHeader(http.StatusOK)
	body := "Thanks for visiting!\n"
	fmt.Fprintf(w, "%s", body)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	paramStr := strings.Split(r.URL.Path, "/")
	if len(paramStr) < 3 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not found: "+r.URL.Path)
		return
	}

	CID := paramStr[2]
	err := deleteEntry(CID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "%s\n", err.Error())
		return
	}

	fmt.Fprintf(w, "%s deleted!\n", CID)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	courses, err := list()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error retrieving course list: %v\n", err)
		return
	}

	for _, course := range courses {
		fmt.Fprintf(w, "%s %s %s\n", course.CID, course.CNAME, course.CPREREQ)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	courses, err := list()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error retrieving course list: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Total entries: %d\n", len(courses))
}

func insertHandler(w http.ResponseWriter, r *http.Request) {
	// Split URL
	paramStr := strings.Split(r.URL.Path, "/")
	if len(paramStr) < 5 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not enough arguments: "+r.URL.Path)
		return
	}

	CID := paramStr[2]
	CNAME := paramStr[3]
	CPREREQ := paramStr[4]

	if CID == "" || CNAME == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "CID and CNAME must be specified.")
		return
	}

	course := initCourse(CID, CNAME, CPREREQ)

	err := insert(course)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Failed to add course: ", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "New record added successfully.\n", course.CID)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	paramStr := strings.Split(r.URL.Path, "/")
	if len(paramStr) < 3 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not found: "+r.URL.Path)
		return
	}

	CID := paramStr[2]
	course, err := search(CID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not found: "+CID)
		return
	}

	fmt.Fprintf(w, "%s %s %s\n", course.CID, course.CNAME, course.CPREREQ)
}
