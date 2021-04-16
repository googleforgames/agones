tool
extends EditorPlugin


func _enter_tree():
	add_custom_type(
		"AgonesSdk",
		"HTTPRequest",
		preload("res://addons/com.google.agones/AgonesSdk.gd"),
		preload("res://addons/com.google.agones/agones.png")
	)
	add_custom_type(
		"AgonesAlpha",
		"HTTPRequest",
		preload("res://addons/com.google.agones/AgonesAlpha.gd"),
		preload("res://addons/com.google.agones/agones.png")
	)
	add_custom_type(
		"AgonesBeta",
		"HTTPRequest",
		preload("res://addons/com.google.agones/AgonesBeta.gd"),
		preload("res://addons/com.google.agones/agones.png")
	)


func _exit_tree():
	remove_custom_type("AgonesSdk")
	remove_custom_type("AgonesAlpha")
	remove_custom_type("AgonesBeta")
