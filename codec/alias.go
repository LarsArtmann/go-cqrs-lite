package codec

import (
	gocodec "github.com/larsartmann/go-codec"
)

// This file re-exports all public symbols from github.com/larsartmann/go-codec
// so that existing imports of github.com/larsartmann/go-cqrs-lite/codec/v4
// continue to work. New code should import go-codec directly.

// --- Types ---

type Encoding = gocodec.Encoding
type Codec = gocodec.Codec
type BufferEncoder = gocodec.BufferEncoder
type CBORCodec = gocodec.CBORCodec
type JSONCodec = gocodec.JSONCodec
type CBORCompactCodec = gocodec.CBORCompactCodec
type RawCodec = gocodec.RawCodec
type COSESign1 = gocodec.COSESign1
type COSEEncrypt0 = gocodec.COSEEncrypt0

// --- Encoding constants ---

const (
	EncodingJSON = gocodec.EncodingJSON
	EncodingCBOR = gocodec.EncodingCBOR
	EncodingRaw  = gocodec.EncodingRaw
)

// --- COSE header parameter labels ---

const (
	COSEHeaderAlg         = gocodec.COSEHeaderAlg
	COSEHeaderCrit        = gocodec.COSEHeaderCrit
	COSEHeaderContentType = gocodec.COSEHeaderContentType
	COSEHeaderKid         = gocodec.COSEHeaderKid
	COSEHeaderIV          = gocodec.COSEHeaderIV
	COSEHeaderPartialIV   = gocodec.COSEHeaderPartialIV
)

// --- COSE algorithm identifiers ---

const (
	COSEAlgHMACSHA256_64    = gocodec.COSEAlgHMACSHA256_64
	COSEAlgHMACSHA256       = gocodec.COSEAlgHMACSHA256
	COSEAlgAES256GCM        = gocodec.COSEAlgAES256GCM
	COSEAlgChaCha20Poly1305 = gocodec.COSEAlgChaCha20Poly1305
	COSEAlgEdDSA            = gocodec.COSEAlgEdDSA
	COSEAlgEd25519          = gocodec.COSEAlgEd25519
)

// --- Error sentinels ---

var (
	ErrUnknownEncoding      = gocodec.ErrUnknownEncoding
	ErrEncodeRawType        = gocodec.ErrEncodeRawType
	ErrDecodeRawType        = gocodec.ErrDecodeRawType
	ErrInvalidCOSESign1     = gocodec.ErrInvalidCOSESign1
	ErrInvalidCOSEEncrypt0  = gocodec.ErrInvalidCOSEEncrypt0
	ErrCOSEAlgorithmOverflow = gocodec.ErrCOSEAlgorithmOverflow
	ErrCOSEInvalidAlgorithm = gocodec.ErrCOSEInvalidAlgorithm
)

// --- Codec lookup and detection ---

var (
	ForEncoding = gocodec.ForEncoding
	AutoDetect  = gocodec.AutoDetect
)

// --- CBOR helpers ---

var (
	CBOREncMode     = gocodec.CBOREncMode
	CBORDecMode     = gocodec.CBORDecMode
	Diagnose        = gocodec.Diagnose
	NewCBOREncoder  = gocodec.NewCBOREncoder
	NewCBORDecoder  = gocodec.NewCBORDecoder
	TranscodeToJSON = gocodec.TranscodeToJSON
	Size            = gocodec.Size
)

// --- Envelope ---

var (
	WrapEncode    = gocodec.WrapEncode
	UnwrapDecode  = gocodec.UnwrapDecode
)

// --- Base64 JSON helpers ---

var (
	DecodeBase64String       = gocodec.DecodeBase64String
	MarshalBase64JSON        = gocodec.MarshalBase64JSON
	MarshalBase64JSONWithModule = gocodec.MarshalBase64JSONWithModule
	UnmarshalBase64JSON      = gocodec.UnmarshalBase64JSON
	AssignBase64JSON         = gocodec.AssignBase64JSON
	WrapCOSEMarshal          = gocodec.WrapCOSEMarshal
)

// --- COSE structure ---

var (
	NormalizeCOSEAlgorithm       = gocodec.NormalizeCOSEAlgorithm
	MarshalCOSEProtectedHeader   = gocodec.MarshalCOSEProtectedHeader
	UnmarshalCOSEProtectedHeader = gocodec.UnmarshalCOSEProtectedHeader
	COSEAlgHeader                = gocodec.COSEAlgHeader
)

// PrepareCOSESetup re-exports [gocodec.PrepareCOSESetup]. Deprecated: use go-codec directly.
func PrepareCOSESetup[Cfg any, O ~func(*Cfg)](cfg *Cfg, opts []O, alg int64) ([]byte, error) {
	return gocodec.PrepareCOSESetup(cfg, opts, alg)
}

var (
	MarshalCOSESign1           = gocodec.MarshalCOSESign1
	UnmarshalCOSESign1         = gocodec.UnmarshalCOSESign1
	MarshalCOSEEncrypt0        = gocodec.MarshalCOSEEncrypt0
	UnmarshalCOSEEncrypt0      = gocodec.UnmarshalCOSEEncrypt0
	SigStructure               = gocodec.SigStructure
	EncStructure0              = gocodec.EncStructure0
	COSESign1String            = gocodec.COSESign1String
	COSEEncrypt0String         = gocodec.COSEEncrypt0String
)
