package adaptor

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"sync"

	"github.com/compose/transporter/pkg/message"
	"github.com/compose/transporter/pkg/pipe"
	"github.com/olivere/elastic"
)

const (
	APPBASE_BUFFER_LEN int = 2000
)

// Appbase is an adaptor to connect a pipeline to
// an Appbase cluster.
type Appbase struct {
	// pull these in from the node
	uri *url.URL

	appName   string
	typename  string
	typeMatch *regexp.Regexp

	pipe *pipe.Pipe
	path string

	client      *elastic.Client
	bulkService *elastic.BulkService
	bulkMutex   *sync.Mutex
	//timerDoneChan chan struct{}
	count    int
	username string
	password string
	debug    bool
	bulkSize int

	running      bool
	bulkBodySize int
}

// NewAppbase creates a new Appbase adaptor.
// Appbase adaptors cannot be used as a source,
func NewAppbase(p *pipe.Pipe, path string, extra Config) (StopStartListener, error) {
	var (
		conf AppbaseConfig
		err  error
	)
	if err = extra.Construct(&conf); err != nil {
		return nil, NewError(CRITICAL, path, fmt.Sprintf("bad config (%s)", err.Error()), nil)
	}

	if conf.URI == "" {
		conf.URI = `https://scalr.api.appbase.io`
	}

	if conf.Namespace == "" {
		return nil, fmt.Errorf("namespace required, but missing ")
	}

	if conf.UserName == "" || conf.Password == "" {
		return nil, fmt.Errorf("both username and password required, but missing ")
	}

	u, err := url.Parse(conf.URI)
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(conf.UserName, conf.Password)

	if conf.BulkSize == 0 {
		conf.BulkSize = 512000 //500kb
	}

	appbase := &Appbase{
		uri:       u,
		pipe:      p,
		bulkMutex: &sync.Mutex{},
		//timerDoneChan: make(chan struct{}),
		bulkSize: conf.BulkSize,
		debug:    conf.Debug,
		username: conf.UserName,
		password: conf.Password,
	}

	appbase.debugLog("Appbase conf: %#v", conf)

	appbase.appName, appbase.typename, err = extra.splitNamespace()
	appbase.typeMatch = regexp.MustCompile(".*")
	if err != nil {
		return appbase, NewError(CRITICAL, path, fmt.Sprintf("can't split namespace into app name and type (%s)", err.Error()), nil)
	}

	return appbase, nil
}

// Start the adaptor as a source (not implemented)
func (a *Appbase) Start() error {
	return fmt.Errorf("appbase can't function as a source")
}

// Listen starts the listener
func (a *Appbase) Listen() error {
	defer a.Stop()

	if err := a.setupClient(); err != nil {
		a.pipe.Err <- NewError(ERROR, a.path, fmt.Sprintf("appbase error (%s)", err), "")
	}

	a.running = true

	return a.pipe.Listen(a.addBulkCommand, a.typeMatch)
}

// Stop the adaptor
func (a *Appbase) Stop() error {
	if a.running {
		a.running = false
		a.pipe.Stop()
		a.commitBulk(true)
		a.debugLog("Documents sent: %d", a.count)
	}
	return nil
}

func (a *Appbase) addBulkCommand(msg *message.Msg) (*message.Msg, error) {
	id, err := msg.IDString("_id")
	if err != nil {
		id = ""
	}

	switch msg.Op {
	case message.Delete:
		bulkRequest := elastic.NewBulkDeleteRequest().Index(a.appName).Type(a.typename).Id(id)
		a.AddBulkRequestSize(bulkRequest)
		a.bulkService.Add(bulkRequest)
		break
	case message.Update:
		bulkRequest := elastic.NewBulkUpdateRequest().Index(a.appName).Type(a.typename).Id(id).Doc(msg.Data)
		a.AddBulkRequestSize(bulkRequest)
		a.bulkService.Add(bulkRequest)
		break
	default:
		bulkRequest := elastic.NewBulkIndexRequest().Index(a.appName).Type(a.typename).Id(id).Doc(msg.Data)
		a.AddBulkRequestSize(bulkRequest)
		a.bulkService.Add(bulkRequest)
		break
	}

	a.commitBulk(true)

	return msg, nil
}

func (a *Appbase) setupClient() error {
	var err error
	a.client, err = elastic.NewClient(
		elastic.SetURL(a.uri.String()),
		elastic.SetSniff(false),
	)

	if err != nil {
		return err
	}

	a.bulkService = a.client.Bulk().Index(a.appName).Type(a.typename)

	return nil

}

func (a *Appbase) commitBulk(commitNow bool) {
	//
	if a.bulkBodySize >= a.bulkSize || a.bulkService.NumberOfActions() >= APPBASE_BUFFER_LEN || commitNow {
		a.debugLog("Appbase: Sending %d documents.", a.bulkService.NumberOfActions())
		a.count += a.bulkService.NumberOfActions()
		a.debugLog("Appbase request size: %d", a.bulkBodySize)

		_, err := a.bulkService.Do()
		if err != nil {
			a.pipe.Err <- NewError(CRITICAL, a.path, fmt.Sprintf("appbase error (%s)", err), nil)
			a.pipe.Stop()
		}
		a.bulkBodySize = 0
		fmt.Println("appbase: DONE")
		//		if bulkResponse.Errors {
		//			for _, item := range bulkResponse.Failed() {
		//				a.pipe.Err <- NewError(ERROR, a.path, fmt.Sprintf("appbase bulk error id:%s (%s)", item.Id, item.Error), nil)
		//			}
		//		}
	}
}

func (a *Appbase) debugLog(format string, v ...interface{}) {
	if a.debug {
		log.Printf(format, v)
	}
}

func (a *Appbase) AddBulkRequestSize(bulkRequest elastic.BulkableRequest) {
	source, err := bulkRequest.Source()
	if err == nil {
		for _, line := range source {
			a.bulkBodySize += len(fmt.Sprintf("%s\n", line))
		}
	}
}

type AppbaseConfig struct {
	URI       string `json:"uri" doc:"the uri to connect to, in the form https://scalr.api.appbase.io`
	UserName  string `json:"username" doc:"appbase application username`
	Password  string `json:"password" doc:"appbase application password`
	Namespace string `json:"namespace" doc:"appbase application name and type to write"`
	Debug     bool   `json:"debug" doc:"display debug information"`
	BulkSize  int    `json:"bulksize" doc:"Define the size of the buffer to bulk operations"`
}
