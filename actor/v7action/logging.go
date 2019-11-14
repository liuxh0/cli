package v7action

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	logcache "code.cloudfoundry.org/log-cache/pkg/client"
	"code.cloudfoundry.org/log-cache/pkg/rpc/logcache_v1"
	"github.com/sirupsen/logrus"

	"code.cloudfoundry.org/cli/actor/loggingaction"
)

const StagingLog = "STG"

var flushInterval = 300 * time.Millisecond

type LogMessage struct {
	message        string
	messageType    string
	timestamp      time.Time
	sourceType     string
	sourceInstance string
}

func (log LogMessage) Message() string {
	return log.message
}

func (log LogMessage) Type() string {
	return log.messageType
}

func (log LogMessage) Staging() bool {
	return log.sourceType == StagingLog
}

func (log LogMessage) Timestamp() time.Time {
	return log.timestamp
}

func (log LogMessage) SourceType() string {
	return log.sourceType
}

func (log LogMessage) SourceInstance() string {
	return log.sourceInstance
}

func NewLogMessage(message string, messageType string, timestamp time.Time, sourceType string, sourceInstance string) *LogMessage {
	return &LogMessage{
		message:        message,
		messageType:    messageType,
		timestamp:      timestamp,
		sourceType:     sourceType,
		sourceInstance: sourceInstance,
	}
}

type LogMessages []*LogMessage

func (lm LogMessages) Len() int { return len(lm) }

func (lm LogMessages) Less(i, j int) bool {
	return lm[i].timestamp.Before(lm[j].timestamp)
}

func (lm LogMessages) Swap(i, j int) {
	lm[i], lm[j] = lm[j], lm[i]
}

type channelWriter struct {
	errChannel chan error
}

func (c channelWriter) Write(bytes []byte) (n int, err error) {
	c.errChannel <- errors.New(strings.Trim(string(bytes), "\n"))

	return len(bytes), nil
}

func GetStreamingLogs(appGUID string, client LogCacheClient) (<-chan LogMessage, <-chan error, context.CancelFunc) {
	logrus.Info("Start Tailing Logs")

	outgoingLogStream := make(chan LogMessage, 1000)
	outgoingErrStream := make(chan error, 1000)
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		defer close(outgoingLogStream)
		defer close(outgoingErrStream)

		logcache.Walk(
			ctx,
			appGUID,
			logcache.Visitor(func(envelopes []*loggregator_v2.Envelope) bool {
				logMessages := convertEnvelopesToLogMessages(envelopes)
				for _, logMessage := range logMessages {
					select {
					case <-ctx.Done():
						return false
					default:
						outgoingLogStream <- *logMessage
					}
				}

				return true
			}),
			client.Read,
			logcache.WithWalkStartTime(time.Now().Add(-5*time.Second)),
			logcache.WithWalkEnvelopeTypes(logcache_v1.EnvelopeType_LOG),
			logcache.WithWalkBackoff(logcache.NewAlwaysRetryBackoff(250*time.Millisecond)),
			logcache.WithWalkLogger(log.New(channelWriter{
				errChannel: outgoingErrStream,
			}, "", 0)),
		)
	}()

	return outgoingLogStream, outgoingErrStream, cancelFunc
}

func convertEnvelopesToLogMessages(envelopes []*loggregator_v2.Envelope) []*LogMessage {
	var logMessages []*LogMessage
	for _, envelope := range envelopes {
		logEnvelope, ok := envelope.GetMessage().(*loggregator_v2.Envelope_Log)
		if !ok {
			continue
		}
		log := logEnvelope.Log

		logMessages = append(logMessages, NewLogMessage(
			string(log.Payload),
			loggregator_v2.Log_Type_name[int32(log.Type)],
			time.Unix(0, envelope.GetTimestamp()),
			envelope.GetTags()["source_type"],
			envelope.GetInstanceId(),
		))
	}
	return logMessages
}

func (actor Actor) GetStreamingLogsForApplicationByNameAndSpace(appName string, spaceGUID string, client LogCacheClient) (<-chan LogMessage, <-chan error, context.CancelFunc, Warnings, error) {
	app, allWarnings, err := actor.GetApplicationByNameAndSpace(appName, spaceGUID)
	if err != nil {
		return nil, nil, nil, allWarnings, err
	}

	messages, logErrs, cancelFunc := GetStreamingLogs(app.GUID, client)

	return messages, logErrs, cancelFunc, allWarnings, err
}

func (actor Actor) GetRecentLogsForApplicationByNameAndSpace(appName string, spaceGUID string, client LogCacheClient) ([]LogMessage, Warnings, error) {
	app, allWarnings, err := actor.GetApplicationByNameAndSpace(appName, spaceGUID)
	if err != nil {
		return nil, allWarnings, err
	}

	logCacheMessages, err := loggingaction.GetRecentLogs(app.GUID, client)
	if err != nil {
		return nil, allWarnings, err
	}

	//TODO: Messages need sorting for most recent?
	// logCacheMessages = client.SortRecent(logCacheMessages)

	var logMessages []LogMessage

	for _, message := range logCacheMessages {
		logMessages = append(logMessages, LogMessage{
			message:        message.Message,
			messageType:    message.MessageType,
			timestamp:      message.Timestamp, // time.Unix(0, message.Timestamp),
			sourceType:     message.SourceType,
			sourceInstance: message.SourceInstance,
		})
	}

	return logMessages, allWarnings, nil
}
