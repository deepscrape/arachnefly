package domain

// Define a struct to hold the validated parameters
type DeleteMachineParams struct {
	MachineID   string `validate:"required,min=5,max=255,hexadecimalan"` // Example validations
	ForceDelete string `validate:"omitempty,eq=true|eq=false"`
}
