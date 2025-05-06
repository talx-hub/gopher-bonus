package model

import "time"

const DefaultTimeout time.Duration = 3 * time.Second
const DefaultWorkerCountMultiplier int = 8
const DefaultRequestCount = 100500

const HeaderContentType = "Content-Type"
