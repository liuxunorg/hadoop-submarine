package submarine

import "time"

// ClientInterface redis client interface
type ClientInterface interface {
	// Close closes the connection.
	Close() error

	// Cmd calls the given Redis command.
	///Cmd(cmd string, args ...interface{}) *redis.Resp

	// PipeAppend adds the given call to the pipeline queue.
	// Use PipeResp() to read the response.
	///PipeAppend(cmd string, args ...interface{})

	// PipeResp returns the reply for the next request in the pipeline queue. Err
	// with ErrPipelineEmpty is returned if the pipeline queue is empty.
	///PipeResp() *redis.Resp

	// PipeClear clears the contents of the current pipeline queue, both commands
	// queued by PipeAppend which have yet to be sent and responses which have yet
	// to be retrieved through PipeResp. The first returned int will be the number
	// of pending commands dropped, the second will be the number of pending
	// responses dropped
	PipeClear() (int, int)

	// ReadResp will read a Resp off of the connection without sending anything
	// first (useful after you've sent a SUSBSCRIBE command). This will block until
	// a reply is received or the timeout is reached (returning the IOErr). You can
	// use IsTimeout to check if the Resp is due to a Timeout
	//
	// Note: this is a more low-level function, you really shouldn't have to
	// actually use it unless you're writing your own pub/sub code
	///ReadResp() *redis.Resp
}

// Client structure representing a client connection to redis
type Client struct {
	commandsMapping map[string]string
	///client          *redis.Client
	client 			ClientInterface
}

// NewClient build a client connection and connect to a redis address
func NewClient(addr string, cnxTimeout time.Duration, commandsMapping map[string]string) (ClientInterface, error) {
	var err error
	c := &Client{
		commandsMapping: commandsMapping,
	}

	// c.client, err = redis.DialTimeout("tcp", addr, cnxTimeout)
	// TODO error!!!!
	return c.client, err
}
