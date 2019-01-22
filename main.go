package main

import (
	"fmt"
	"time"

	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type WorkerPool struct {
}

type Job struct {
	Info string
}

func (t Job) Do(workID int) {
	// go func() {
	// if workID == 0 {
	fmt.Println("job start:", workID, t.Info)
	// }
	time.Sleep(5 * time.Second)
	// }()

}

type Worker struct {
	ID         int
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
}

func (w Worker) Start() {
	go func() {
		for {
			// fmt.Println("worker start")
			w.WorkerPool <- w.JobChannel

			// fmt.Println("worker in")

			select {
			case job := <-w.JobChannel:
				job.Do(w.ID)

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

func NewWorker(workerPool chan chan Job, id int) Worker {
	return Worker{
		ID:         id,
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool)}
}

var JobQueue chan Job

type Dispatcher struct {
	WorkerPool chan chan Job
	maxWorkers int
}

func NewDispatcher(maxWorkers, maxJobs int) *Dispatcher {
	pool := make(chan chan Job, maxJobs)
	return &Dispatcher{WorkerPool: pool, maxWorkers: maxWorkers}
}

func (d *Dispatcher) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.WorkerPool, i)
		worker.Start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {

		select {
		case job := <-JobQueue:

			go func(job Job) {
				jobChannel := <-d.WorkerPool

				jobChannel <- job
			}(job)
		}
	}
}

func JobAdd(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)

	info := r.FormValue("info")
	work := Job{Info: info}
	JobQueue <- work
	fmt.Fprintf(w, "%v", "0")
}

func main() {

	dispatcher := NewDispatcher(100, 10)
	dispatcher.Run()

	JobQueue = make(chan Job)
	var router = mux.NewRouter()
	router.HandleFunc("/test", JobAdd).Methods("POST")

	http.Handle("/", router)

	http.ListenAndServe(":9090", handlers.CORS(handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"POST", "GET", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With"}))(router))

	return
}
