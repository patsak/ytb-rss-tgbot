package ytbrss


type Entity struct {
	User int64
}
type AudioEntity struct {
	Entity
	Audio string
	Title string
}

type Rss struct {
	Dest string
}

func (r *Rss) AppendEntity(e *AudioEntity) error {
	return nil
}
