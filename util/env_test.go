package util

import (
    "testing"
)

func TestEnvSubset(t *testing.T) {
    env := MakeEnv([]string{"TEST1=test", "TEST2=test"})

    // env
    env2 := env

    if !env.Subset(env2) {
        t.Errorf("equal identical: %#v %#v", env, env2)
    }

    env2 = MakeEnv([]string{"TEST2=test", "TEST1=test"})
    if !env.Subset(env2) {
        t.Errorf("equal reordered: %#v %#v", env, env2)
    }

    env2 = MakeEnv([]string{})
    if env.Subset(env2) {
        t.Errorf("non-equal empty: %#v %#v", env, env2)
    }

    env2 = MakeEnv([]string{"TEST1=test2"})
    if env.Subset(env2) {
        t.Errorf("non-equal short: %#v %#v", env, env2)
    }

    env2 = env
    env2 = MakeEnv([]string{"TEST1=test", "TEST2=test2"})
    if env.Subset(env2) {
        t.Errorf("non-equal modified: %#v %#v", env, env2)
    }

    env2 = env
    env2 = MakeEnv([]string{"TEST1=test", "TEST2=test", "TEST3=test"})
    if !env.Subset(env2) {
        t.Errorf("equal superset: %#v %#v", env, env2)
    }

    env2 = env
    env2 = MakeEnv([]string{"TEST1=test", "TEST1b=test", "TEST2=test"})
    if !env.Subset(env2) {
        t.Errorf("equal superset: %#v %#v", env, env2)
    }

    env2 = env
    env2 = MakeEnv([]string{"TEST1=test", "TEST2=test2", "TEST3=test"})
    if env.Subset(env2) {
        t.Errorf("non-equal modified superset: %#v %#v", env, env2)
    }
}
