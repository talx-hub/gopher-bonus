package model

import "time"

const DefaultTimeout = 500 * time.Millisecond
const DefaultWorkerCountMultiplier = 2
const DefaultRequestCount = 100500
const DefaultChannelCapacity = 1024

const WatcherTickTimeout = 3 * time.Second

const HeaderContentType = "Content-Type"

type ContextKey string

const KeyContextLogger ContextKey = "logger"
const KeyContextUserID ContextKey = "userID"

const KeyLoggerError = "error"
