namespace go abcoder.testdata.thrifts.person
namespace java abcoder.testdata.thrifts.person

include "../gender/gender.thrift"

struct Person {
	1: string name
	2: gender.Gender gender
}