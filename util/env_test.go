package util

import (
    "testing"
)

var testEnvSubset = []struct{
    env     Env
    other   Env
    subset  bool
}{
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST2=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST2=test", "TEST1=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv(),
        false,
    },
    {
        MakeEnv(),
        MakeEnv("TEST1=test", "TEST2=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test"),
        false,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST2=test"),
        false,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test2", "TEST2=test"),
        false,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST2=test2"),
        false,
    },
    // XXX: match duplicates?
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST2=test", "TEST2=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST1=test", "TEST2=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST2=test", "TEST3=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST1b=test", "TEST2=test"),
        true,
    },
    {
        MakeEnv("TEST1=test", "TEST2=test"),
        MakeEnv("TEST1=test", "TEST2=test2", "TEST3=test"),
        false,
    },

}

func TestEnvSubset(t *testing.T) {
    for _, test := range testEnvSubset {
        subset := test.env.Subset(test.other)

        if subset == test.subset {

        } else if !test.subset {
            t.Errorf("should not be a subset:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        } else {
            t.Errorf("should be a subset:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        }
    }
}
