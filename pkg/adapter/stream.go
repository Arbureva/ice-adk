package adapter

// TODO: 能否有一些 Encoder 和 Decoder 方法？比如 (Encode|Decode)ResoningAdapter 之类的，专门用于将常用类型转换成 ChunkMessageAdapter ?
type ChunkMessageAdapter struct {
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}
