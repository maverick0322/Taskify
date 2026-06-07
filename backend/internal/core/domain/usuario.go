package domain

import (
	"errors"
	"strings"
	"time"
)

// Constantes para evitar "Magic Numbers" en nuestras validaciones
const (
	minNombreLength   = 2
	minPasswordLength = 8
)

// Errores de dominio específicos
var (
	ErrNombreInvalido   = errors.New("domain: el nombre o apellidos no cumplen la longitud mínima")
	ErrEmailInvalido    = errors.New("domain: el formato del email es inválido")
	ErrPasswordInvalida = errors.New("domain: la contraseña no cumple con los requisitos de seguridad")
	ErrUsuarioMenorEdad = errors.New("domain: el usuario debe tener al menos 18 años")
)

// DatosPersonales es un Value Object que encapsula la información del perfil.
// Al ser un Value Object, debe ser inmutable tras su creación.
type DatosPersonales struct {
	nombre          string
	apellidos       string
	fechaNacimiento time.Time
}

// NewDatosPersonales actúa como un Factory Method para garantizar que el Value Object
// nazca en un estado válido. (Programación defensiva)
func NewDatosPersonales(nombre, apellidos string, fechaNac time.Time) (DatosPersonales, error) {
	if len(strings.TrimSpace(nombre)) < minNombreLength || len(strings.TrimSpace(apellidos)) < minNombreLength {
		return DatosPersonales{}, ErrNombreInvalido
	}

	// Validación de edad (Ejemplo de regla de negocio pura)
	edadMinima := time.Now().AddDate(-18, 0, 0)
	if fechaNac.After(edadMinima) {
		return DatosPersonales{}, ErrUsuarioMenorEdad
	}

	return DatosPersonales{
		nombre:          strings.TrimSpace(nombre),
		apellidos:       strings.TrimSpace(apellidos),
		fechaNacimiento: fechaNac,
	}, nil
}

// Usuario representa la Entidad raíz (Aggregate Root) en nuestro dominio.
type Usuario struct {
	id              string
	email           string
	passwordHash    string
	datosPersonales DatosPersonales
}

// NewUsuario inicializa una entidad Usuario asegurando su validez.
func NewUsuario(id, email, passwordHash string, perfil DatosPersonales) (*Usuario, error) {
	if strings.TrimSpace(id) == "" {
		// Preferimos panic solo si es un error fatal de desarrollo (ej. UUID no inyectado),
		// pero devolvemos error normal para control de flujo de usuario.
		return nil, errors.New("domain: el ID de usuario no puede estar vacío")
	}
	
	if !strings.Contains(email, "@") {
		// Validación básica. Se asume que el Regex validado se hizo en una capa superior o Value Object de Email.
		return nil, ErrEmailInvalido
	}

	if len(passwordHash) < minPasswordLength {
		return nil, ErrPasswordInvalida
	}

	return &Usuario{
		id:              id,
		email:           email,
		passwordHash:    passwordHash,
		datosPersonales: perfil,
	}, nil
}

// Getters: Go no usa la palabra get. Exponemos el estado sin permitir modificación directa
// para mantener el encapsulamiento y proteger las invariantes (OCP).

func (u *Usuario) ID() string {
	return u.id
}

func (u *Usuario) Email() string {
	return u.email
}

func (u *Usuario) Perfil() DatosPersonales {
	return u.datosPersonales
}