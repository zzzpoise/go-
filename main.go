package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string
const (
	TaskPending  TaskStatus = "排队等待"
	TaskRunning  TaskStatus = "正在运行"
	TaskSuccess  TaskStatus = "已完成"
	TaskFailed   TaskStatus = "运行失败"
	TaskCanceled TaskStatus = "已取消"
)

type Task struct {
	ID         string
	Name       string
	Status     TaskStatus
	Handler    func() error
	Timeout    time.Duration
	RetryTimes int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	err        error
	ctx        context.Context
	cancel     context.CancelFunc
}

type TaskManager struct {
	maxWorker int
	taskMap   map[string]*Task
	taskChan  chan *Task
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

func NewTaskManager(maxWork int) *TaskManager {
	manager := &TaskManager{
		maxWorker: maxWork,
		taskMap:   make(map[string]*Task),
		taskChan:  make(chan *Task, 100),
		stopChan:  make(chan struct{}),
	}
	for i := 0; i < maxWork; i++ {
		go manager.worker()
	}
	return manager
}

// 工人循环干活函数
func (m *TaskManager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.stopChan:
			return
		case task, ok := <-m.taskChan:
			if !ok {
				return
			}
			m.runSingleTask(task)
		}
	}
}

// 工人处理单个任务完整流程
func (m *TaskManager) runSingleTask(task *Task) {
	m.mu.Lock()
	task.Status = TaskRunning
	task.UpdatedAt = time.Now()
	m.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			m.mu.Lock()
			task.err = fmt.Errorf("任务崩溃：%v", r)
			task.Status = TaskFailed
			task.UpdatedAt = time.Now()
			m.mu.Unlock()
		}
	}()

	ctx, cancel := context.WithTimeout(task.ctx, task.Timeout)
	defer cancel()
	errChan := make(chan error, 1)
	go func() {
		errChan <- task.Handler()
	}()

	select {
	case <-ctx.Done():
		task.err = fmt.Errorf("任务超时")
	case err := <-errChan:
		task.err = err
	}

	m.mu.Lock()
	if task.err == nil {
		task.Status = TaskSuccess
	} else {
		if task.RetryTimes > 0 {
			task.RetryTimes--
			task.Status = TaskPending
			m.taskChan <- task
		} else {
			task.Status = TaskFailed
		}
	}
	task.UpdatedAt = time.Now()
	m.mu.Unlock()
}

func (m *TaskManager) SubmitTask(name string, handler func() error, timeout time.Duration, retry int) (string, error) {
	taskID := uuid.NewString()
	taskCtx, taskCancel := context.WithCancel(context.Background())
	task := &Task{
		ID:         taskID,
		Name:       name,
		Status:     TaskPending,
		Handler:    handler,
		Timeout:    timeout,
		RetryTimes: retry,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ctx:        taskCtx,
		cancel:     taskCancel,
	}
	m.mu.Lock()
	m.taskMap[taskID] = task
	m.mu.Unlock()
	m.taskChan <- task
	return taskID, nil
}

func (m *TaskManager) GetTask(taskID string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, exist := m.taskMap[taskID]
	if !exist {
		return nil, fmt.Errorf("任务不存在")
	}
	return task, nil
}

func (m *TaskManager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
	close(m.taskChan)
	fmt.Println("任务结束，管理器关闭完成")
}

func main() {
	manager := NewTaskManager(2)
	defer manager.Stop()

	task1Func := func() error {
		fmt.Println("任务1开始运行")
		time.Sleep(2 * time.Second)
		fmt.Println("任务1执行完毕")
		return nil
	}
	id1, _ := manager.SubmitTask("正常任务", task1Func, 5*time.Second, 1)
	fmt.Println("提交任务1，编号：", id1)

	task2Func := func() error {
		fmt.Println("任务2运行，主动报错")
		return fmt.Errorf("任务运行出错")
	}
	id2, _ := manager.SubmitTask("报错重试任务", task2Func, 3*time.Second, 2)
	fmt.Println("提交任务2，编号：", id2)

	time.Sleep(6 * time.Second)
	t1, _ := manager.GetTask(id1)
	t2, _ := manager.GetTask(id2)
	fmt.Printf("\n任务1最终状态：%s，错误：%v\n", t1.Status, t1.err)
	fmt.Printf("任务2最终状态：%s，错误：%v\n", t2.Status, t2.err)
}