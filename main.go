package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var tmpl *template.Template
var db *sql.DB

type Task struct {
	Id   int
	Task string
	Done bool
}

func initDB() {
	var err error

	// Initialize db variable
	db, err = sql.Open("sqlite3", "go_htmx_learn.db")
	if err != nil {
		log.Fatal(err)
	}

	// Check the database connection
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	fmt.Println("Setting up server...")

	// Setup SQLite
	initDB()
	defer db.Close()

	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")

	// v1
	v1 := r.Group("/v1")
	{
		v1.GET("/", Homepage)
		v1.GET("/tasks", fetchTasks)
		v1.GET("/newtaskform", getTaskForm)
		v1.POST("/tasks", addTask)
		v1.GET("/gettaskupdateform/:id", getTaskUpdateForm)
		v1.PUT("/tasks/:id", updateTask)
		v1.POST("/tasks/:id", updateTask)
		v1.DELETE("/tasks/:id", deleteTask)
	}

	port := "4000"
	fmt.Println("Starting server on http://localhost:4000")
	r.Run(":" + port)
}

func Homepage(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", nil)
}

func fetchTasks(c *gin.Context) {
	todos, err := getTasks(db)
	if err != nil {
		log.Fatal(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	//fmt.Println(todos)

	//If you used "define" to define the template, use the name you gave it here, not the filename
	c.HTML(http.StatusOK, "todoList", todos)
}

func getTaskForm(c *gin.Context) {
	c.HTML(http.StatusOK, "addTaskForm", nil)
}

func addTask(c *gin.Context) {
	task := c.PostForm("task")

	fmt.Println(task)

	query := "INSERT INTO todos (task, done) VALUES (?, ?)"

	stmt, err := db.Prepare(query)

	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, executeErr := stmt.Exec(task, 0)

	if executeErr != nil {
		log.Fatal(executeErr)
	}

	// Return a new list of Todos
	todos, _ := getTasks(db)

	//You can also just send back the single task and append it
	//I like returning the whole list just to get everything fresh, but this might not be the best strategy
	c.HTML(http.StatusOK, "todoList", todos)
}

func getTaskUpdateForm(c *gin.Context) {
	//Convert string id from URL to integer
	taskId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := getTaskByID(db, taskId)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "updateTaskForm", task)
}

func updateTask(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	taskItem := c.PostForm("task")
	var taskStatus bool

	fmt.Println(c.PostForm("done"))

	//Check the string value of the checkbox
	switch strings.ToLower(c.PostForm("done")) {
	case "yes", "on":
		taskStatus = true
	case "no", "off":
		taskStatus = false
	default:
		taskStatus = false
	}

	task := Task{
		taskId, taskItem, taskStatus,
	}

	updateErr := updateTaskById(db, task)

	if updateErr != nil {
		log.Fatal(updateErr)
	}

	//Refresh all Tasks
	todos, _ := getTasks(db)

	c.HTML(http.StatusOK, "todoList", todos)
}

func deleteTask(c *gin.Context) {
	taskId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	err = deleTaskWithID(db, taskId)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	//Return list
	todos, _ := getTasks(db)

	c.HTML(http.StatusOK, "todoList", todos)
}

func getTasks(dbPointer *sql.DB) ([]Task, error) {
	query := "SELECT id, task, done FROM todos"

	rows, err := dbPointer.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var todo Task
		rowErr := rows.Scan(&todo.Id, &todo.Task, &todo.Done)
		if rowErr != nil {
			return nil, err
		}
		tasks = append(tasks, todo)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil

}

func getTaskByID(dbPointer *sql.DB, id int) (*Task, error) {
	query := "SELECT id, task, done FROM todos WHERE id = ?"

	var task Task
	row := dbPointer.QueryRow(query, id)
	err := row.Scan(&task.Id, &task.Task, &task.Done)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no task was found with task %d", id)
		}
		return nil, err
	}

	return &task, nil

}

func updateTaskById(dbPointer *sql.DB, task Task) error {
	query := "UPDATE todos SET task = ?, done = ? WHERE id = ?"

	result, err := dbPointer.Exec(query, task.Task, task.Done, task.Id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		fmt.Println("No rows updated")
	} else {
		fmt.Printf("%d row(s) updated\n", rowsAffected)
	}

	return nil

}

func deleTaskWithID(dbPointer *sql.DB, id int) error {
	query := "DELETE FROM todos WHERE id = ?"

	stmt, err := dbPointer.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	result, err := stmt.Exec(id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no task found with id %d", id)
	}

	fmt.Printf("Deleted %d task(s)\n", rowsAffected)
	return nil

}
