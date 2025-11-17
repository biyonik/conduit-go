// -----------------------------------------------------------------------------
// Database Types - SQL Builder İçin Yardımcı Tipler
// -----------------------------------------------------------------------------
// Bu dosya, QueryBuilder'ın kullandığı internal struct tiplerini içerir.
// OrderClause, JoinClause gibi yapılar burada tanımlanır. Bu sayede
// SQL injection gibi güvenlik açıklarına karşı daha güvenli bir yapı oluşturulur.
//
// Örneğin OrderClause, direction alanını enum gibi kullanarak sadece
// "ASC" ve "DESC" değerlerini kabul eder. Bu sayede kullanıcı input'u
// direkt SQL'e enjekte edilemez.
// -----------------------------------------------------------------------------

package database

// OrderDirection, ORDER BY için izin verilen yönleri temsil eder.
// Bu enum-like yapı sayesinde SQL injection riski ortadan kalkar.
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// OrderClause, bir ORDER BY ifadesini güvenli bir şekilde temsil eder.
// Column ve Direction alanları ayrı tutularak, derleme zamanında
// SQL string'inin güvenli bir şekilde oluşturulması sağlanır.
//
// Alanlar:
//   - Column: Sıralama yapılacak kolon adı (backtick ile sarmalanacak)
//   - Direction: Sıralama yönü (sadece ASC veya DESC olabilir)
//
// Örnek Kullanım:
//
//	OrderClause{Column: "created_at", Direction: OrderDesc}
//	→ SQL: ORDER BY `created_at` DESC
type OrderClause struct {
	Column    string
	Direction OrderDirection
}

// WhereClause, bir WHERE koşulunu güvenli bir şekilde temsil eder.
// Tüm değerler placeholder (?) olarak kullanılır, bu sayede
// prepared statement'lar ile SQL injection korunması sağlanır.
//
// Alanlar:
//   - Column: Koşul uygulanacak kolon adı
//   - Operator: Karşılaştırma operatörü (=, <, >, LIKE, vb.)
//   - Value: Karşılaştırılacak değer (prepared statement'a bağlanır)
//   - Boolean: Önceki koşulla bağlantı tipi ("AND" veya "OR")
//
// Güvenlik Notu:
// Bu yapı sayesinde tüm değerler prepared statement'lar ile bağlanır.
// Operator whitelist kontrolü Grammar katmanında yapılır.
type WhereClause struct {
	Column   string
	Operator string
	Value    interface{}
	Boolean  string // "AND" veya "OR"
}

// JoinType, JOIN tiplerini temsil eden enum-like yapıdır.
type JoinType string

const (
	InnerJoin JoinType = "INNER"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
	CrossJoin JoinType = "CROSS"
)

// JoinClause, bir JOIN ifadesini güvenli bir şekilde temsil eder.
// İleride JOIN desteği eklendiğinde bu yapı kullanılacaktır.
//
// Alanlar:
//   - Type: JOIN tipi (INNER, LEFT, RIGHT, CROSS)
//   - Table: JOIN yapılacak tablo adı
//   - First: İlk kolon (örn: "users.id")
//   - Operator: Karşılaştırma operatörü (genellikle "=")
//   - Second: İkinci kolon (örn: "posts.user_id")
//
// Örnek Kullanım:
//
//	JoinClause{
//	    Type: LeftJoin,
//	    Table: "posts",
//	    First: "users.id",
//	    Operator: "=",
//	    Second: "posts.user_id",
//	}
//	→ SQL: LEFT JOIN `posts` ON `users`.`id` = `posts`.`user_id`
type JoinClause struct {
	Type     JoinType
	Table    string
	First    string
	Operator string
	Second   string
}
