extends Control

@onready var status: Label = $Marker/Content/Status

func _ready() -> void:
	status.text = "Placeholder loaded from Godot"
