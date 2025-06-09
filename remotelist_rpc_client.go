package main

import (
	"fmt"
	"math/rand"
	"net/rpc"
)

func main() {
	client, err := rpc.Dial("tcp", ":5000")

	if err != nil {
		fmt.Print("dialing:", err)
	}

	for i := 0; i < 5; i++ {
		fmt.Printf("\n Iteração %d\n", i+1)

		// 1. Criar nova lista
		var listID int
		err = client.Call("RemoteList.AddList", struct{}{}, &listID)
		fmt.Printf("Lista criada com ID: %d\n", listID)

		// 2. Adicionar 5 valores aleatórios
		for j := 0; j < 5; j++ {
			valor := rand.Intn(100)
			var reply bool
			err = client.Call("RemoteList.Append", [2]int{listID, valor}, &reply)
			fmt.Printf("Valor %d adicionado à lista %d\n", valor, listID)
		}

		// 3. Mostrar size da lista
		var size uint32
		err = client.Call("RemoteList.Size", listID, &size)
		fmt.Printf("Tamanho atual da lista %d: %d\n", listID, size)

		// 4. Mostrar Get no último índice
		var valorUltimo int
		err = client.Call("RemoteList.Get", [2]int{listID, int(size - 1)}, &valorUltimo)
		fmt.Printf("Último valor (index %d): %d\n", size-1, valorUltimo)

		// 5. Remover último valor
		var removido int
		err = client.Call("RemoteList.Remove", listID, &removido)
		fmt.Printf("Valor removido: %d\n", removido)

		// 6. Mostrar novo último valor (se houver)
		err = client.Call("RemoteList.Size", listID, &size)
		var novoUltimo int
		err = client.Call("RemoteList.Get", [2]int{listID, int(size - 1)}, &novoUltimo)
		fmt.Printf("Novo último valor (index %d): %d\n", size-1, novoUltimo)

		// 7. Mostrar size final
		fmt.Printf("Tamanho final da lista %d: %d\n", listID, size)

	}
}
