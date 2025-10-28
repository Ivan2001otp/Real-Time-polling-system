package utils

// session status
var ACTIVE string = "active"
var DRAFT string = "draft"
var CLOSED string = "closed"

// question types
var SINGLE string = "single"
var MULTIPLE string = "multiple"

// jwt
var JWT_CLAIM_ISSUER  string = "polling-platform";
var AUTHORIZATION_HEADER string  = "Authorization";

// redis constants
var USER_KEY_PREFIX string = "user:email:";
var REDIS_CONNECTION string = "redis://localhost:6379";

// mongo constants
var USERS_COLLECTION string = "organizers";
var SESSION_COLLECTION string = "session_polls";
var DB_NAME string = "polling";
var MONGO_CONNECTION string = "mongodb://localhost:27017";
var VOTES_COLLECTION string = "votes";

// kafka constants
const (
	VOTES_SUBMITTED_TOPIC = "votes.submitted"
	VOTES_PROCESSED_TOPIC = "votes.processed"
	RESULTS_UPDATED_TOPIC = "votes.updated"
)

var KAFKA_CONNECTION string = "localhost:29092";
var KAFKA_GROUP_ID string = "vote-processor-group";
var KAFKA_RESULTS_BROADCASTER_GROUP string = "results-broadcaster-group";
