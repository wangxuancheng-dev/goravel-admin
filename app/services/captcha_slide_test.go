package services

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestCaptchaService(kind string) *CaptchaServiceImpl {
	return &CaptchaServiceImpl{ctx: context.Background(), kind: kind}
}

func TestNormalizeCaptchaType(t *testing.T) {
	require.Equal(t, CaptchaTypeImage, NormalizeCaptchaType(""))
	require.Equal(t, CaptchaTypeImage, NormalizeCaptchaType("image"))
	require.Equal(t, CaptchaTypeImage, NormalizeCaptchaType("bogus"))
	require.Equal(t, CaptchaTypeSlide, NormalizeCaptchaType(" Slide "))
}

func TestVerifySlideAnswer(t *testing.T) {
	require.True(t, verifySlideAnswer(100, "100"))
	require.True(t, verifySlideAnswer(100, "103,22"))
	require.True(t, verifySlideAnswer(100, "95.4"))
	require.False(t, verifySlideAnswer(100, "106"))
	require.False(t, verifySlideAnswer(100, "abc"))
	require.False(t, verifySlideAnswer(100, "NaN"))
	require.False(t, verifySlideAnswer(100, ""))
}

func TestSlideCaptchaGenerateAndVerify(t *testing.T) {
	svc := newTestCaptchaService(CaptchaTypeSlide)

	challenge, err := svc.GenerateChallenge()
	require.NoError(t, err)
	require.Equal(t, CaptchaTypeSlide, challenge.Type)
	require.NotEmpty(t, challenge.ID)
	require.NotEmpty(t, challenge.MasterImage)
	require.NotEmpty(t, challenge.TileImage)
	require.Positive(t, challenge.TileWidth)
	require.Positive(t, challenge.TileHeight)

	answer := PeekCaptchaAnswer(challenge.ID)
	require.NotEmpty(t, answer)
	x, err := strconv.Atoi(answer)
	require.NoError(t, err)

	ok, key := svc.Verify(challenge.ID, answer)
	require.True(t, ok, key)

	// One attempt only: a verified captcha cannot be replayed.
	ok, key = svc.Verify(challenge.ID, answer)
	require.False(t, ok)
	require.Equal(t, "captcha_expired", key)

	// A wrong answer fails and also burns the captcha.
	challenge, err = svc.GenerateChallenge()
	require.NoError(t, err)
	x, err = strconv.Atoi(PeekCaptchaAnswer(challenge.ID))
	require.NoError(t, err)
	ok, key = svc.Verify(challenge.ID, strconv.Itoa(x+slideTolerancePx+20))
	require.False(t, ok)
	require.Equal(t, "captcha_invalid", key)
	ok, key = svc.Verify(challenge.ID, strconv.Itoa(x))
	require.False(t, ok)
	require.Equal(t, "captcha_expired", key)
}

func TestCaptchaTypesAreNotInterchangeable(t *testing.T) {
	slideSvc := newTestCaptchaService(CaptchaTypeSlide)
	imageSvc := newTestCaptchaService(CaptchaTypeImage)

	challenge, err := slideSvc.GenerateChallenge()
	require.NoError(t, err)
	answer := PeekCaptchaAnswer(challenge.ID)
	ok, key := imageSvc.Verify(challenge.ID, answer)
	require.False(t, ok)
	require.Equal(t, "captcha_invalid", key)

	imgChallenge, err := imageSvc.GenerateChallenge()
	require.NoError(t, err)
	require.Equal(t, CaptchaTypeImage, imgChallenge.Type)
	require.NotEmpty(t, imgChallenge.Image)
	imgAnswer := PeekCaptchaAnswer(imgChallenge.ID)
	ok, key = slideSvc.Verify(imgChallenge.ID, imgAnswer)
	require.False(t, ok)
	require.Equal(t, "captcha_invalid", key)
}
