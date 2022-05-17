package featureflag

import (
	"errors"
	"fmt"

	"github.com/gomodule/redigo/redis"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/redispool"
)

var (
	pool = redispool.Store
)

func GetEvaluatedFlagSetFromCache(flags []*FeatureFlag, a *actor.Actor) FlagSet {
	flagSet := FlagSet{}

	c := pool.Get()
	defer c.Close()

	visitorID, err := getVisitorIDForActor(a)

	if err != nil {
		return flagSet
	}

	for _, flag := range flags {
		value, _ := redis.Bool(c.Do("HGET", getFlagCacheKey(flag.Name), visitorID))

		flagSet[flag.Name] = value
	}

	return flagSet
}

func SetEvaluatedFlagToCache(name string, a *actor.Actor, value bool) {
	c := pool.Get()
	defer c.Close()

	var visitorID string

	visitorID, err := getVisitorIDForActor(a)

	if err != nil {
		return
	}

	c.Do("HSET", getFlagCacheKey(name), visitorID, fmt.Sprintf("%v", value))
}

func ClearFlagFromCache(name string) {
	c := pool.Get()
	defer c.Close()

	c.Do("DEL", getFlagCacheKey(name))
}

func getVisitorIDForActor(a *actor.Actor) (string, error) {
	if a.IsAuthenticated() {
		return fmt.Sprintf("uid_%v", a.UID), nil
	} else if a.AnonymousUID != "" {
		return "auid_" + a.AnonymousUID, nil
	} else {
		return "", errors.New("UID/AnonymousUID are emptry for the given actor.")
	}
}

func getFlagCacheKey(name string) string {
	return "ff_" + name
}