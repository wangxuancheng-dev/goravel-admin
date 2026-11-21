package services

import (
	"goravel/app/models"

	"goravel-payment"
)

// paymentOrderFromModel maps the host payment row into the reusable gateway DTO.
func paymentOrderFromModel(pay *models.Payment) *payment.Order {
	if pay == nil {
		return nil
	}
	return &payment.Order{
		PaymentNo:    pay.PaymentNo,
		Amount:       pay.Amount,
		Status:       pay.Status,
		Remark:       pay.Remark,
		ThirdPartyNo: pay.ThirdPartyNo,
		PayTime:      pay.PayTime,
	}
}
