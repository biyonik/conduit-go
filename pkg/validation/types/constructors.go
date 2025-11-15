package types

// Array, yeni bir ArrayType nesnesi oluşturur.
//
// Döndürür:
//   - *ArrayType: Yeni ArrayType örneği
func Array() *ArrayType {
	return &ArrayType{}
}

// Boolean, yeni bir BooleanType nesnesi oluşturur.
//
// Döndürür:
//   - *BooleanType: Yeni BooleanType örneği
func Boolean() *BooleanType {
	return &BooleanType{}
}

// CreditCard, yeni bir CreditCardType nesnesi oluşturur.
func CreditCard() *CreditCardType {
	return &CreditCardType{}
}

// Date, yeni bir DateType nesnesi oluşturur.
//
// Döndürür:
//   - *DateType: Yeni DateType örneği
func Date() *DateType {
	return &DateType{
		format: "2006-01-02", // Varsayılan tarih formatı
	}
}

// Number, yeni bir NumberType nesnesi oluşturur.
func Number() *NumberType {
	return &NumberType{}
}

// Object, yeni bir ObjectType nesnesi oluşturur.
func Object() *ObjectType {
	return &ObjectType{}
}

// String, yeni bir StringType nesnesi oluşturur.
//
// Döndürür:
//   - *StringType: Yeni StringType örneği
func String() *StringType {
	return &StringType{}
}

// Uuid, yeni bir UuidType nesnesi oluşturur.
//
// Döndürür:
//   - *UuidType: Yeni UuidType örneği
func Uuid() *UuidType {
	return &UuidType{version: 0}
}

func AdvancedString() *AdvancedStringType {
	return &AdvancedStringType{}
}

func Iban() *IbanType {
	return &IbanType{}
}
