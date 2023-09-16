package acr128s

import (
	"fmt"
	"time"

	"github.com/dumacp/smartcard"
	"github.com/dumacp/smartcard/nxp/mifare"
)

type Reader struct {
	smartcard.IReader
	mifare.IReaderClassic
	dev        *Device
	readerName string
	slot       Slot
	seq        int
}

// NewReader Create New Reader interface
func NewReader(dev *Device, readerName string, slot Slot) *Reader {
	r := &Reader{
		dev:        dev,
		readerName: readerName,
		slot:       slot,
	}
	return r
}

func (r *Reader) Transmit(apdu []byte) ([]byte, error) {

	header := BuildHeader__PC_to_RDR_XfrBlock(r.seq, r.slot, len(apdu))

	data, err := BuildFrame(header, apdu)

	if err != nil {
		return nil, err
	}
	r.seq += 1

	response, err := r.dev.SendRecv(data, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if len(response) <= 4 {

		if err := VerifyStatusReponse(response); err != nil {
			return nil, err
		}
		response, err = r.dev.SendRecv(FRAME_NACK, 100*time.Millisecond)
		if err != nil {
			return nil, err
		}
	}

	dataResponse, err := GetResponse__RDR_to_PC_DataBlock(response)
	if err != nil {
		return nil, err
	}

	return dataResponse, nil
}

func (r *Reader) EscapeCommand(apdu []byte) ([]byte, error) {

	header := BuildHeader__PC_to_RDR_Escape(r.seq, SLOT_PICC, len(apdu))
	data, err := BuildFrame(header, apdu)

	if err != nil {
		return nil, err
	}

	response, err := r.dev.SendRecv(data, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}
	r.seq += 1

	if len(response) <= 4 {
		if err := VerifyStatusReponse(response); err != nil {
			return nil, err
		}
		response, err = r.dev.SendRecv(FRAME_NACK, 100*time.Millisecond)
		if err != nil {
			return nil, err
		}
	}

	dataResponse, err := GetResponse__RDR_to_PC_Escape(response)
	if err != nil {
		return nil, err
	}

	return dataResponse, nil
}

func (r *Reader) IccPowerOff() ([]byte, error) {

	header := BuildHeader__PC_to_RDR_IccPowerOff(r.seq, r.slot)

	data, err := BuildFrame(header, nil)

	if err != nil {
		return nil, err
	}
	r.seq += 1

	response, err := r.dev.SendRecv(data, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if len(response) <= 4 {

		if err := VerifyStatusReponse(response); err != nil {
			return nil, err
		}
		response, err = r.dev.SendRecv(FRAME_NACK, 100*time.Millisecond)
		if err != nil {
			return nil, err
		}
	}

	dataResponse, err := GetResponse__RDR_to_PC_SlotStatus(response)
	if err != nil {
		return nil, err
	}

	return dataResponse, nil
}

func (r *Reader) IccPowerOn() ([]byte, error) {

	header := BuildHeader__PC_to_RDR_IccPowerOn(r.seq, r.slot)

	data, err := BuildFrame(header, nil)

	if err != nil {
		return nil, err
	}
	r.seq += 1

	response, err := r.dev.SendRecv(data, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if len(response) <= 4 {

		if err := VerifyStatusReponse(response); err != nil {
			return nil, err
		}
		response, err = r.dev.SendRecv(FRAME_NACK, 100*time.Millisecond)
		if err != nil {
			return nil, err
		}
	}

	dataResponse, err := GetResponse__RDR_to_PC_SlotStatus(response)
	if err != nil {
		return nil, err
	}

	return dataResponse, nil
}

// Create New Card interface
func (r *Reader) ConnectCard() (smartcard.ICard, error) {
	respEscape, err := r.EscapeCommand([]byte{0xE0, 0, 0, 0x25, 0})
	if err != nil {
		return nil, fmt.Errorf("connect card err = %s, %w", err, smartcard.ErrComm)
	}
	if respEscape[len(respEscape)-1] == 0 {
		return nil, fmt.Errorf("without card")
	}

	respGetData, err := r.Transmit([]byte{0xFF, 0xCA, 0, 0, 0})
	if err != nil {
		return nil, fmt.Errorf("without card")
	}

	if err := mifare.VerifyResponseIso7816(respGetData); err != nil {
		return nil, err
	}

	uid := respGetData[:len(respGetData)-2]

	// fmt.Printf("getData response: [% X]\n", respGetData)

	cardS := &Card{
		uid:    uid,
		reader: r,
	}
	return cardS, nil
}

// Create New Card interface
func (r *Reader) ConnectSamCard() (smartcard.ICard, error) {
	if _, err := r.EscapeCommand([]byte{0xE0, 0, 0, 0x2E, 2, 0, 10}); err != nil {
		return nil, fmt.Errorf("connect card err = %s, %w", err, smartcard.ErrComm)
	}
	// fmt.Printf("readExtraGuardTime response: [% X]\n", respReadExtraGuardTime)

	respIccPowerOn, err := r.IccPowerOn()
	if err != nil {
		return nil, fmt.Errorf("connect card err = %s, %w", err, smartcard.ErrComm)
	}
	// fmt.Printf("iccPowerOn response: [% X]\n", respIccPowerOn)

	// if err := mifare.VerifyResponseIso7816(respIccPowerOn); err != nil {
	// 	return nil, err
	// }
	if len(respIccPowerOn) < 1 || respIccPowerOn[0] != 0x3B {
		return nil, fmt.Errorf("bad response: [% X]", respIccPowerOn)
	}

	atr := respIccPowerOn[:]

	cardS := &Card{
		atr:    atr,
		reader: r,
	}
	return cardS, nil
}