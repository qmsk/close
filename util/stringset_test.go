package util

import (
    "testing"
)

var testStringSet = []struct{
    env     StringSet
    other   StringSet
    subset  bool
    equals  bool
}{
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST2=test"),
        true, true,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST2=test", "TEST1=test"),
        true, true,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet(),
        false, false,
    },
    {
        MakeStringSet(),
        MakeStringSet("TEST1=test", "TEST2=test"),
        true, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test"),
        false, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST2=test"),
        false, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test2", "TEST2=test"),
        false, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST2=test2"),
        false, false,
    },
    // XXX: match duplicates?
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST2=test", "TEST2=test"),
        true, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST1=test", "TEST2=test"),
        true, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST2=test", "TEST3=test"),
        true, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST1b=test", "TEST2=test"),
        true, false,
    },
    {
        MakeStringSet("TEST1=test", "TEST2=test"),
        MakeStringSet("TEST1=test", "TEST2=test2", "TEST3=test"),
        false, false,
    },

}

func TestStringSetEquals(t *testing.T) {
    for _, test := range testStringSet {
        equals := test.env.Equals(test.other)

        if equals == test.equals{

        } else if !test.equals{
            t.Errorf("should not be equal:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        } else {
            t.Errorf("should be equal:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        }
    }
}
func TestStringSetSubset(t *testing.T) {
    for _, test := range testStringSet {
        subset := test.env.Subset(test.other)

        if subset == test.subset {

        } else if !test.subset {
            t.Errorf("should not be a subset:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        } else {
            t.Errorf("should be a subset:\nSelf:\t%v\nOther:\t%v\n", test.env, test.other)
        }
    }
}
