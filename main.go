package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

var cpool ConstantPool
var bytes []byte
var byteIndex int = 0

type ExceptionTable struct {
	start_pc   u2
	end_pc     u2
	handler_pc u2
	catch_type u2
}

type CodeAttribute struct {
	attribute_name_index   u2
	attribute_length       u4
	max_stqck              u2
	max_locals             u2
	code_length            u4
	code                   []byte
	exception_table_length u2
	exception_tables       []ExceptionTable
	attributes_count       u2
	attribute_infos        []AttributeInfo
}

type AttributeInfo struct {
	attribute_name_index u2
	attribute_length     u4
	body                 []byte
}

type MethodInfo struct {
	access_flags     u2
	name_index       u2
	descriptor_index u2
	attributes_count u2
	ai               []CodeAttribute
}

type CONSTANT_Class_info struct {
	tag        u1
	name_index u2
}

type CONSTANT_Fieldref_info struct {
	tag                 u1
	class_index         u2
	name_and_type_index u2
}

type CONSTANT_Methodref_info struct {
	tag                 u1
	class_index         u2
	name_and_type_index u2
}

type CONSTANT_String_info struct {
	tag          u1
	string_index u2
}

type CONSTANT_NameAndType_info struct {
	tag              u1
	name_index       u2
	descriptor_index u2
}

type CONSTANT_Utf8_info struct {
	tag    u1
	length u2
	bytes  []byte
}

func readCafebabe() [4]byte {
	byteIndex += 4
	return [4]byte{bytes[0], bytes[1], bytes[2], bytes[3]}
}

type u1 byte
type u2 uint16
type u4 uint32

func readU2() u2 {
	left := bytes[byteIndex]
	right := bytes[byteIndex+1]
	byteIndex += 2

	return u2(u2(left)*256 + u2(right))
}
func readU4() u4 {
	b1 := bytes[byteIndex]
	b2 := bytes[byteIndex+1]
	b3 := bytes[byteIndex+2]
	b4 := bytes[byteIndex+3]
	byteIndex += 4

	return u4(u4(b1)*256*256*256 + u4(b2)*256*256 + u4(b3)*256 + u4(b4))
}

func readBytes(n int) []byte {
	r := bytes[byteIndex : byteIndex+n]
	byteIndex += n
	return r
}

func readByte() u1 {
	b := bytes[byteIndex]
	byteIndex++
	return u1(b)
}

func readAttributeInfo() AttributeInfo {
	a := AttributeInfo{
		attribute_name_index: readU2(),
		attribute_length:     readU4(),
	}
	a.body = readBytes(int(a.attribute_length))
	return a
}

func readExceptionTable() {
	readU2()
	readU2()
	readU2()
	readU2()
}

func readCodeAttribute() CodeAttribute {
	a := CodeAttribute{
		attribute_name_index: readU2(),
		attribute_length:     readU4(),
		max_stqck:            readU2(),
		max_locals:           readU2(),
		code_length:          readU4(),
	}
	a.code = readBytes(int(a.code_length))
	a.exception_table_length = readU2()
	for i := u2(0); i < a.exception_table_length; i++ {
		readExceptionTable()
	}
	a.attributes_count = readU2()
	for i := u2(0); i < a.attributes_count; i++ {
		readAttributeInfo()
	}
	return a
}

func readMethodInfo() MethodInfo {
	methodInfo := MethodInfo{
		access_flags:     readU2(),
		name_index:       readU2(),
		descriptor_index: readU2(),
		attributes_count: readU2(),
	}
	var cas []CodeAttribute
	for i := u2(0); i < methodInfo.attributes_count; i++ {
		ca := readCodeAttribute()
		cas = append(cas, ca)
	}
	methodInfo.ai = cas

	return methodInfo
}

type ConstantPool []interface{}

// https://docs.oracle.com/javase/specs/jvms/se11/html/jvms-4.html#jvms-4.1
type ClassFile struct {
	magic               [4]byte
	minor_version       u2
	major_version       u2
	constant_pool_count u2
	constant_pool       ConstantPool
	access_flags        u2
	this_class          u2
	super_class         u2
	interface_count     u2
	fields_count        u2
	methods_count       u2
	methods             []MethodInfo
	attributes_count    u2
	attributes          []AttributeInfo
}

func parseClassFile(filename string) *ClassFile {
	var err error
	bytes, err = ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	magic := readCafebabe()
	minor_version := readU2()
	major_version := readU2()
	constant_pool_count := readU2()

	var constant_pool []interface{}
	constant_pool = append(constant_pool, nil)
	for i := u2(0); i < constant_pool_count-1; i++ {
		tag := readByte()
		//fmt.Printf("[i=%d] tag=%02X\n", i, tag)
		var e interface{}
		switch tag {
		case 0x09:
			e = &CONSTANT_Fieldref_info{
				class_index:         readU2(),
				name_and_type_index: readU2(),
				tag:                 tag,
			}
		case 0x0a:
			e = &CONSTANT_Methodref_info{
				class_index:         readU2(),
				name_and_type_index: readU2(),
				tag:                 tag,
			}
		case 0x08:
			e = &CONSTANT_String_info{
				string_index: readU2(),
				tag:          tag,
			}
		case 0x07:
			e = &CONSTANT_Class_info{
				name_index: readU2(),
				tag:        tag,
			}
		case 0x01:
			ln := readU2()
			e = &CONSTANT_Utf8_info{
				tag:    tag,
				length: ln,
				bytes:  readBytes(int(ln)),
			}
		case 0x0c:
			e = &CONSTANT_NameAndType_info{
				name_index:       readU2(),
				descriptor_index: readU2(),
				tag:              tag,
			}
		default:
			panic("unknown tag")
		}
		//e.tag = tag
		constant_pool = append(constant_pool, e)
	}

	access_flags := readU2()
	this_class := readU2()
	super_class := readU2()
	interface_count := readU2()
	fields_count := readU2()
	methods_count := readU2()

	var methods []MethodInfo = make([]MethodInfo, methods_count)
	for i := u2(0); i < methods_count; i++ {
		methodInfo := readMethodInfo()
		methods[i] = methodInfo
	}
	attributes_count := readU2()
	var attributes []AttributeInfo
	for i := u2(0); i < attributes_count; i++ {
		attr := readAttributeInfo()
		attributes = append(attributes, attr)
	}
	if len(bytes) == byteIndex {
		fmt.Printf("__EOF__\n")
	}

	return &ClassFile{
		magic:               magic,
		minor_version:       minor_version,
		major_version:       major_version,
		constant_pool_count: constant_pool_count,
		constant_pool:       constant_pool,
		access_flags:        access_flags,
		this_class:          this_class,
		super_class:         super_class,
		interface_count:     interface_count,
		fields_count:        fields_count,
		methods_count:       methods_count,
		methods:             methods,
		attributes_count:    attributes_count,
		attributes:          attributes,
	}
}

func c2s(c interface{}) string {
	switch c.(type) {
	case *CONSTANT_Fieldref_info:
		cf := c.(*CONSTANT_Fieldref_info)
		return fmt.Sprintf("Fieldref\t#%d.#%d",
			cf.class_index, cf.name_and_type_index)
	case *CONSTANT_Methodref_info:
		cm := c.(*CONSTANT_Methodref_info)
		return fmt.Sprintf("Methodref\t#%d.#%d",
			cm.class_index, cm.name_and_type_index)
	case *CONSTANT_Class_info:
		return fmt.Sprintf("Class\t%d", c.(*CONSTANT_Class_info).name_index)
	case *CONSTANT_String_info:
		return fmt.Sprintf("String\t%d", c.(*CONSTANT_String_info).string_index)
	case *CONSTANT_NameAndType_info:
		cn := c.(*CONSTANT_NameAndType_info)
		return fmt.Sprintf("NameAndType\t#%d:#%d", cn.name_index, cn.descriptor_index)
	case *CONSTANT_Utf8_info:
		return fmt.Sprintf("Utf8\t%s", c.(*CONSTANT_Utf8_info).bytes)
	default:
		panic("Unknown constant pool")
	}

}

func debugConstantPool(cp []interface{})  {
	for i, c := range cp {
		if i == 0 {
			continue
		}
		fmt.Printf(" #%02d = ",i)
		s := c2s(c)
		fmt.Printf("%s\n", s)
	}
}

func (cp ConstantPool) get(i u2) interface{} {
	return cp[i]
}

func (cp ConstantPool) getFieldref(id u2) *CONSTANT_Fieldref_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_Fieldref_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getMethodref(id u2) *CONSTANT_Methodref_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_Methodref_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getClassInfo(id u2) *CONSTANT_Class_info {
	entry := cp.get(id)
	ci, ok := entry.(*CONSTANT_Class_info)
	if !ok {
		panic("type mismatch")
	}
	return ci
}

func (cp ConstantPool) getNameAndType(id u2) *CONSTANT_NameAndType_info {
	entry := cp.get(id)
	c, ok := entry.(*CONSTANT_NameAndType_info)
	if !ok {
		panic("type mismatch")
	}
	return c
}

func (cp ConstantPool) getUTF8Byttes(id u2) []byte {
	entry := cp.get(id)
	utf8, ok := entry.(*CONSTANT_Utf8_info)
	if !ok {
		panic("type mismatch")
	}
	return utf8.bytes
}

func debugClassFile(cf *ClassFile) {
	for _, char := range cf.magic {
		fmt.Printf("%x ", char)
	}

	fmt.Printf("\n")
	fmt.Printf("major_version = %d, minior_version = %d\n", cf.major_version, cf.minor_version)
	fmt.Printf("access_flags=%d\n", cf.access_flags)
	ci := cf.constant_pool.getClassInfo(cf.this_class)
	fmt.Printf("class %s\n", cf.constant_pool.getUTF8Byttes(ci.name_index))
	fmt.Printf("  super_class=%d\n", cf.super_class)

	fmt.Printf("Constant pool:\n")
	debugConstantPool(cf.constant_pool)

	fmt.Printf("interface_count=%d\n", cf.interface_count)
	//fmt.Printf("interfaces=%d\n", interfaces)
	fmt.Printf("fields_count=%d\n", cf.fields_count)
	fmt.Printf("methods_count=%d\n", cf.methods_count)

	for _, methodInfo := range cf.methods{
		methodName := cf.constant_pool.getUTF8Byttes(methodInfo.name_index)
		fmt.Printf(" %s:\n", methodName)
		for _, ca  := range methodInfo.ai {
			for _, c := range ca.code {
				fmt.Printf(" %02x", c)
			}
		}
		fmt.Printf("\n")
	}
	fmt.Printf("attributes_count=%d\n", cf.attributes_count)
	fmt.Printf("attribute=%v\n", cf.attributes[0])
}

func getByte() byte {
	b := bytes[byteIndex]
	byteIndex++
	return b
}

var stack []interface{}

func push(e interface{}) {
	stack = append(stack, e)
}

func pop() interface{} {
	e := stack[len(stack)-1]
	newStack := stack[0:len(stack)-1]
	stack = newStack
	return e
}

func executeCode(code []byte) {
	fmt.Printf("len code=%d\n", len(code))

	byteIndex = 0
	bytes = code
	for _, b := range code {
		fmt.Printf("0x%x ", b)
	}
	fmt.Printf("\n")
	for {
		if byteIndex >= len(bytes) {
			break
		}
		b := getByte()
		fmt.Printf("inst 0x%02x\n", b)
		switch b {
		case 0x12: // ldc
			operand := readByte()
			fmt.Printf("  ldc 0x%02x\n", operand)
			push(operand)
		case 0xb1: // return
			fmt.Printf("  return\n")
		case 0xb2: // getstatic
			operand := readU2()
			fmt.Printf("  getstatic 0x%02x\n", operand)
			fieldref := cpool.getFieldref(operand)
			cls := cpool.getClassInfo(fieldref.class_index)
			className := cpool.getUTF8Byttes(cls.name_index)
			nameAndType := cpool.getNameAndType(fieldref.name_and_type_index)
			name := cpool.getUTF8Byttes(nameAndType.name_index)
			desc := cpool.getUTF8Byttes(nameAndType.descriptor_index)
			fmt.Printf("   => %s#%s#%s#%s\n", c2s(fieldref), className, name, desc)
			push(operand)
		case 0xb6: // invokevirtual
			operand := readU2()
			fmt.Printf("  invokevirtual 0x%02x\n", operand)
			methodRef := cpool.getMethodref(operand)
			nameAndType := cpool.getNameAndType(methodRef.name_and_type_index)
			desc := cpool.getUTF8Byttes(nameAndType.descriptor_index)
			desc_args := strings.Split(string(desc), ";")
			num_args := len(desc_args) - 1
			fmt.Printf("  desc=%s, num_args=%d\n", desc, num_args)
			arg0id := pop()
			arg0int := arg0id.(u1)
			arg0 := cpool.get(u2(arg0int))
			fmt.Printf("  arg0=%s\n", c2s(arg0))
		default:
			panic("Unknown instruction")
		}
		fmt.Printf("  stack=%#v\n", stack)

	}
}

func (methodInfo MethodInfo) invoke() {
	for _, ca := range methodInfo.ai {
		executeCode(ca.code)
		fmt.Printf("---\n")
	}
}

func main() {
	cf := parseClassFile("HelloWorld.class")
	cpool = cf.constant_pool
	for _, methodInfo := range cf.methods {
		methodName := cf.constant_pool.getUTF8Byttes(methodInfo.name_index)
		if string(methodName) == "main" {
			methodInfo.invoke()
		}
	}
	debugClassFile(cf)
}

