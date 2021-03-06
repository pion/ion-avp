// +build libvpx

package elements

import (
	"sync"
	"time"

	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	"github.com/xlab/libvpx-go/vpx"
)

// Decoder instance
type Decoder struct {
	sync.Mutex
	Node
	ctx   *vpx.CodecCtx
	iface *vpx.CodecIface
	fps   float32
	typ   int
	run   bool
	async bool
}

// NewDecoder instance. Decoder takes as input VPX streams
// and decodes it into a YCbCr image.
func NewDecoder(fps float32, outType int) *Decoder {
	dec := &Decoder{
		ctx: vpx.NewCodecCtx(),
		typ: outType,
		fps: fps,
	}

	return dec
}

func (dec *Decoder) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		payload := sample.Payload.([]byte)

		if !dec.run {
			videoKeyframe := (payload[0]&0x1 == 0)

			if !videoKeyframe {
				return nil
			}
			dec.run = true

			if dec.fps > 0 {
				dec.async = true
				go dec.producer(dec.fps)
			}
		}

		if dec.iface == nil {
			dec.iface = vpx.DecoderIfaceVP8()
			err := vpx.Error(vpx.CodecDecInitVer(dec.ctx, dec.iface, nil, 0, vpx.DecoderABIVersion))
			if err != nil {
				log.Errorf("%s", err)
			}
		}

		dec.Lock()
		err := vpx.Error(vpx.CodecDecode(dec.ctx, string(payload), uint32(len(payload)), nil, 0))
		dec.Unlock()
		if err != nil {
			return err
		}

		if !dec.async {
			return dec.write()
		}
	}

	return nil
}

func (dec *Decoder) Close() {
	dec.run = false
	dec.Node.Close()
}

func (dec *Decoder) write() error {
	dec.Lock()
	defer dec.Unlock()
	var iter vpx.CodecIter
	img := vpx.CodecGetFrame(dec.ctx, &iter)

	for img != nil {
		img.Deref()

		if dec.typ == TypeYCbCr {
			return dec.Node.Write(&avp.Sample{
				Type:    TypeYCbCr,
				Payload: img.ImageYCbCr(),
			})
		} else if dec.typ == TypeRGBA {
			return dec.Node.Write(&avp.Sample{
				Type:    TypeRGBA,
				Payload: img.ImageRGBA(),
			})
		}
	}
	return nil
}

func (dec *Decoder) producer(fps float32) {
	ticker := time.NewTicker(time.Duration((1/fps)*1000) * time.Millisecond)
	for range ticker.C {
		if !dec.run {
			return
		}

		err := dec.write()
		if err != nil {
			log.Errorf("%s", err)
		}
	}
}
