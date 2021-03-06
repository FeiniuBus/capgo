package mysql

import (
	"database/sql"

	cap "github.com/FeiniuBus/capgo"
)

// LockedMessage ...
type LockedMessage struct {
	message            interface{}
	messageType        int32
	dbConnection       *sql.DB
	dbTransaction      *sql.Tx
	logger             cap.ILogger
	storageTransaction *MySqlStorageTransaction
}

// NewLockedMessage ...
func NewLockedMessage(message interface{}, messageType int32, dbConnection *sql.DB, dbTransaction *sql.Tx, capOptions *cap.CapOptions) cap.ILockedMessage {
	lockedMessage := &LockedMessage{
		message:       message,
		messageType:   messageType,
		dbConnection:  dbConnection,
		dbTransaction: dbTransaction,
		storageTransaction: &MySqlStorageTransaction{
			Options:       capOptions,
			DbConnection:  dbConnection,
			DbTransaction: dbTransaction,
		},
	}
	lockedMessage.logger = cap.GetLoggerFactory().CreateLogger(lockedMessage)
	return lockedMessage
}

// GetMessage ...
func (message *LockedMessage) GetMessage() interface{} {
	return message.message
}

// GetMessageType ...
func (message *LockedMessage) GetMessageType() int32 {
	return message.messageType
}

// Prepare ...
func (message *LockedMessage) Prepare(query string) (stmt interface{}, err error) {
	statement, err := message.dbTransaction.Prepare(query)
	if err != nil {
		message.logger.Log(cap.LevelError, "[Prepare]"+err.Error())
		return nil, err
	}
	return statement, nil
}

// Commit ...
func (message *LockedMessage) Commit() error {
	err := message.dbTransaction.Commit()
	if err != nil {
		message.logger.Log(cap.LevelError, err.Error())
		return err
	}
	return nil
}

// Rollback ...
func (message *LockedMessage) Rollback() error {
	err := message.dbTransaction.Rollback()
	if err != nil {
		message.logger.Log(cap.LevelError, err.Error())
		return err
	}
	return nil
}

// Dispose ...
func (message *LockedMessage) Dispose() {
	err := message.dbConnection.Close()
	if err != nil {
		message.logger.Log(cap.LevelError, "[Dispose]"+err.Error())
	}
}

func (message *LockedMessage) logError(err string) {
	message.logger.LogData(cap.LevelError, "[Enqueue]"+err,
		struct {
			MessageType int32
			Message     interface{}
		}{MessageType: message.messageType, Message: message.message})
}

// ChangeState ...
func (message *LockedMessage) ChangeState(state cap.IState) error {
	stateChanger := cap.NewStateChanger()
	var err error
	if message.messageType == 0 {
		err = stateChanger.ChangePublishedMessage(message.message.(*cap.CapPublishedMessage), state, message.storageTransaction)
	} else if message.messageType == 1 {
		err = stateChanger.ChangeReceivedMessageState(message.message.(*cap.CapReceivedMessage), state, message.storageTransaction)
	} else {
		err = cap.NewCapError("Unknown MessageType.")
	}
	if err != nil {
		message.logError(err.Error())
		return err
	}
	return nil
}
