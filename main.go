//Ivan Gualotuña
package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const estudiante = "Iván Gualotuña"

type Sucursal struct {
	Nombre string
	Demora time.Duration
	Ventas []float64
}

var sucursales = []Sucursal{
	{"Centro", 400 * time.Millisecond, []float64{120.50, 89.90, 202.10}},
	{"Norte", 600 * time.Millisecond, []float64{150.00, 239.20}},
	{"Cumbayá", 800 * time.Millisecond, nil},
	{"Sur", 1000 * time.Millisecond, []float64{75.80, 200.00}},
}

type Consolidado struct {
	mu         sync.Mutex
	Total      float64
	Reportadas int
}

// Sumar agrega el total de una sucursal y el candado deja pasar a una sola goroutine a la vez
func (c *Consolidado) Sumar(total float64) {
	c.mu.Lock()         // si otra goroutine está sumando, esta espera aquí
	defer c.mu.Unlock() // el candado se abre al salir de la función
	c.Total += total
	c.Reportadas++
}

// consultar pide las ventas a una sucursal y devuelve el total y la venta mayor
func consultar(s Sucursal) (float64, float64) {
	time.Sleep(s.Demora) 
	mayor := s.Ventas[0] 
	total := 0.0
	for _, v := range s.Ventas {
		total += v
		if v > mayor {
			mayor = v
		}
	}
	return total, mayor
}

// cierre consulta las sucursales a la vez y escribe el reporte del dia en w 
func cierre(w io.Writer) time.Duration {
	inicio := time.Now()
	var wg sync.WaitGroup
	var c Consolidado
	lineas := make([]string, len(sucursales)) 
	var secuencia time.Duration

	for i, s := range sucursales {
		secuencia += s.Demora
		wg.Add(1) 
		go func(i int, s Sucursal) {
			defer wg.Done() 
			defer func() {  
				if r := recover(); r != nil {
					lineas[i] = fmt.Sprintf("%s: sin datos - fallo recuperado: %v", s.Nombre, r)
				}
			}()
			total, mayor := consultar(s)
			c.Sumar(total)
			lineas[i] = fmt.Sprintf("%s: %d ventas, total $%.2f, venta mayor $%.2f", s.Nombre, len(s.Ventas), total, mayor)
		}(i, s)
	}

	wg.Wait() 
	duracion := time.Since(inicio)

	fmt.Fprintln(w, "Cierre del día - Farmacias del Valle")
	fmt.Fprintln(w, "Programado por:", estudiante)
	fmt.Fprintln(w)
	for _, l := range lineas {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Sucursales reportadas: %d de %d\n", c.Reportadas, len(sucursales))
	fmt.Fprintf(w, "Total del día: $%.2f\n", c.Total)
	fmt.Fprintf(w, "Tiempo de respuesta: %.1f s (en secuencia: %.1f s)\n", duracion.Seconds(), secuencia.Seconds())
	return duracion
}

// verCierre atiende la ruta /cierre: escribe el reporte en la pagina y anota cuanto tard0
func verCierre(w http.ResponseWriter, r *http.Request) {
	duracion := cierre(w)
	fmt.Printf("Consulta atendida en %.1f s\n", duracion.Seconds())
}

// main publica el reporte en http://localhost:8080/cierre y espera solicitudes.
func main() {
	http.HandleFunc("/cierre", verCierre)
	fmt.Println("Servidor de Farmacias del Valle en http://localhost:8080/cierre")
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		fmt.Println("No se pudo iniciar el servidor:", err)
	}
}