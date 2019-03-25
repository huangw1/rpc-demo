package main

type T struct {
	A int
	B string
}

func main() {
	// context scene
	//ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	//defer cancel()
	//
	//select {
	//case <-time.After(1 * time.Second):
	//	fmt.Println("time.After")
	//case <-ctx.Done():
	//	fmt.Println(ctx.Err())
	//}

	// reflect scene
	//t := T{2019, "go"}
	//tt := reflect.TypeOf(t)
	//ttt := reflect.ValueOf(t)
	//fmt.Printf("tt name: %v\n", tt.Name())
	//fmt.Printf("ttt value: %v\n", ttt)
	//fmt.Printf("tt type: %v\n", tt)
	//ttp := reflect.TypeOf(&t)
	//fmt.Printf("ttp type: %v\n", ttp)
	//// 要设置t的值，需要传入t的地址，而不是t的拷贝。
	//// reflect.ValueOf(&t)只是一个地址的值，不是settable, 通过.Elem()解引用获取t本身的reflect.Value
	//s := reflect.ValueOf(&t).Elem()
	//typeOfT := s.Type()
	//for i := 0; i < s.NumField(); i++ {
	//	f := s.Field(i)
	//	fmt.Printf("%d: %s %s = %v\n", i,
	//		typeOfT.Field(i).Name, f.Type(), f.Interface())
	//}

}
