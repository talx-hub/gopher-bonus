package model

import "time"

const DefaultTimeout = 500 * time.Millisecond
const DefaultWorkerCountMultiplier = 8
const DefaultRequestCount = 100500

const HeaderContentType = "Content-Type"

type ContextKey string

const KeyContextLogger ContextKey = "logger"
