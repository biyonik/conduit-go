package controllers

import (
	"net/http"
	"reflect"

	conduitReq "github.com/biyonik/conduit-go/internal/http/request"
	conduitRes "github.com/biyonik/conduit-go/internal/http/response"
	"github.com/biyonik/conduit-go/internal/jobs"
	"github.com/biyonik/conduit-go/pkg/container"
	"github.com/biyonik/conduit-go/pkg/queue"
)

// ExampleQueueController, queue kullanım örneği.
type ExampleQueueController struct {
	Queue queue.Queue
}

// NewExampleQueueController, controller oluşturur.
func NewExampleQueueController(c *container.Container) (*ExampleQueueController, error) {
	queueDriver := c.MustGet(reflect.TypeOf((*queue.Queue)(nil)).Elem()).(queue.Queue)

	return &ExampleQueueController{
		Queue: queueDriver,
	}, nil
}

// SendWelcomeEmail, hoş geldin email'i queue'ya ekler.
//
// POST /api/queue/send-welcome-email
// Body: {"email": "user@example.com", "name": "John Doe"}
func (ec *ExampleQueueController) SendWelcomeEmail(w http.ResponseWriter, r *conduitReq.Request) {
	var reqData struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := r.ParseJSON(&reqData); err != nil {
		conduitRes.Error(w, 400, "Geçersiz JSON formatı")
		return
	}

	// Email job oluştur
	emailJob := jobs.NewSendEmailJob(
		reqData.Email,
		"Welcome to Conduit-Go",
		"Hello "+reqData.Name+"! Welcome to our platform.",
	)

	// Queue'ya ekle
	if err := ec.Queue.Push(emailJob, "emails"); err != nil {
		conduitRes.Error(w, 500, "Job queue'ya eklenemedi")
		return
	}

	conduitRes.Success(w, 200, map[string]interface{}{
		"message": "Email job queued successfully",
		"job_id":  emailJob.GetID(),
	}, nil)
}
