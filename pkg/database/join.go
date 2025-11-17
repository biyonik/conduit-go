package database

// -----------------------------------------------------------------------------
// JOIN OPERATIONS
// -----------------------------------------------------------------------------
// Basit join yapıları için yer ayrılmıştır. Bu demo sürümde Join mantığı
// skeleton olarak bırakılmıştır. İleride grammar destekli JOIN string
// üretimi eklenecektir.
// -----------------------------------------------------------------------------

//// JoinClause, ileride JOIN'lerin tutulacağı yapıdır.
//type JoinClause struct {
//	Type  string // INNER, LEFT, RIGHT, CROSS
//	Table string
//	On    string // basit on ifadesi (ör: `users.id = posts.user_id`)
//}

// Not: Bu sürümde join'ler QueryBuilder.orders içine dahil edilmemiştir.
// Geliştirme aşamasında join desteği eklenecektir.
