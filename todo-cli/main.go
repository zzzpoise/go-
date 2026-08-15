package main

//定义任务数据模型
import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// 实现数据持久化
const dataFile = "tasks.json"

func loadTasks() ([]Task, error) {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil //文件不存在时返回空切片
		}
		return nil, err
	}
	if len(data) == 0 {
		return []Task{}, nil

	}
	var tasks []Task
	err = json.Unmarshal(data, &tasks)
	return tasks, err

}

// 将任务保存到json文件
func saveTasks(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile, data, 0644)
}

// 实现添加任务
func addTask(title string) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	//计算新id：取当前最大id+1
	maxID := 0
	for _, t := range tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}
	newTask := Task{
		ID:        maxID + 1,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}
	tasks = append(tasks, newTask)
	return saveTasks(tasks)
}

// 实现列出任务
func listTasks() error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	for _, t := range tasks {
		status := " "
		if t.Done {
			status = "x"
		}
		fmt.Printf("[%s] %d. %s\n", status, t.ID, t.Title)
	}
	return nil
}

//实现完成和删除任务

func completeTask(id int) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	found := false
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Done = true
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("任务ID %d 不存在", id)
	}
	return saveTasks(tasks)
}

// 过滤掉指定ID的任务
func deleteTask(id int) error {
	tasks, err := loadTasks()
	if err != nil {
		return err
	}
	newTasks := make([]Task, 0)
	for _, t := range tasks {
		if t.ID != id {
			newTasks = append(newTasks, t)
		}
	}
	if len(newTasks) == len(tasks) {
		return fmt.Errorf("任务ID %d 不存在", id)
	}
	return saveTasks(newTasks)
}

// 使用flag解析命令参数
func main() {
	add := flag.String("add", "", "添加任务")
	list := flag.Bool("list", false, "列出任务")
	done := flag.Int("done", 0, "完成任务ID")
	del := flag.Int("del", 0, "删除任务ID")
	flag.Parse()

	switch {
	case *add != "":
		if err := addTask(*add); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)

		}
		fmt.Println("任务已添加")
	case *list:
		if err := listTasks(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)

		}
	case *done != 0:
		if err := completeTask(*done); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("任务已完成")
	case *del != 0:
		if err := deleteTask(*del); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("任务已删除")
	default:
		flag.Usage()
	}
}
