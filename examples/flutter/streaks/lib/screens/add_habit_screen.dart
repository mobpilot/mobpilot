import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/habit.dart';
import '../providers/habits_provider.dart';

class AddHabitScreen extends StatefulWidget {
  final Habit? editHabit;

  const AddHabitScreen({super.key, this.editHabit});

  @override
  State<AddHabitScreen> createState() => _AddHabitScreenState();
}

class _AddHabitScreenState extends State<AddHabitScreen> {
  final _nameController = TextEditingController();
  HabitIcon _selectedIcon = HabitIcon.custom;
  HabitFrequency _selectedFrequency = HabitFrequency.daily;
  String _selectedColor = '#6C63FF';

  static const _colors = [
    '#6C63FF',
    '#FF6B6B',
    '#4ECDC4',
    '#FFE66D',
    '#95E1D3',
    '#F38181',
    '#AA96DA',
    '#A8D8EA',
    '#FF9A8B',
    '#88D8B0',
  ];

  bool get _isEditing => widget.editHabit != null;

  @override
  void initState() {
    super.initState();
    if (widget.editHabit != null) {
      _nameController.text = widget.editHabit!.name;
      _selectedIcon = widget.editHabit!.icon;
      _selectedFrequency = widget.editHabit!.frequency;
      _selectedColor = widget.editHabit!.color;
    }
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_nameController.text.trim().isEmpty) return;

    final habits = context.read<HabitsProvider>();

    if (_isEditing) {
      await habits.updateHabit(widget.editHabit!.copyWith(
        name: _nameController.text.trim(),
        icon: _selectedIcon,
        frequency: _selectedFrequency,
        color: _selectedColor,
      ));
    } else {
      await habits.addHabit(Habit(
        name: _nameController.text.trim(),
        icon: _selectedIcon,
        frequency: _selectedFrequency,
        color: _selectedColor,
      ));
    }

    if (mounted) Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEditing ? 'Edit Habit' : 'New Habit'),
        actions: [
          TextButton(
            onPressed: _save,
            child: const Text('Save'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Name
          TextFormField(
            controller: _nameController,
            autofocus: !_isEditing,
            decoration: const InputDecoration(
              labelText: 'Habit name',
              hintText: 'e.g. Morning Run, Read 30min, Meditate',
              border: OutlineInputBorder(),
            ),
            textCapitalization: TextCapitalization.sentences,
          ),
          const SizedBox(height: 24),

          // Icon picker
          Text('Icon', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: HabitIcon.values.map((icon) {
              final isSelected = icon == _selectedIcon;
              return GestureDetector(
                onTap: () => setState(() => _selectedIcon = icon),
                child: Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: isSelected
                        ? theme.colorScheme.primaryContainer
                        : theme.colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(12),
                    border: isSelected
                        ? Border.all(color: theme.colorScheme.primary, width: 2)
                        : null,
                  ),
                  child: Center(
                    child: Text(icon.emoji, style: const TextStyle(fontSize: 24)),
                  ),
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: 24),

          // Color picker
          Text('Color', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: _colors.map((hex) {
              final color = _hexToColor(hex);
              final isSelected = hex == _selectedColor;
              return GestureDetector(
                onTap: () => setState(() => _selectedColor = hex),
                child: Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: color,
                    shape: BoxShape.circle,
                    border: isSelected
                        ? Border.all(
                            color: theme.colorScheme.onSurface, width: 3)
                        : null,
                  ),
                  child: isSelected
                      ? const Icon(Icons.check, color: Colors.white, size: 20)
                      : null,
                ),
              );
            }).toList(),
          ),
          const SizedBox(height: 24),

          // Frequency
          Text('Frequency', style: theme.textTheme.titleSmall),
          const SizedBox(height: 8),
          RadioGroup<HabitFrequency>(
            groupValue: _selectedFrequency,
            onChanged: (v) => setState(() => _selectedFrequency = v!),
            child: Column(
              children: HabitFrequency.values
                  .where((f) => f != HabitFrequency.custom)
                  .map((freq) => RadioListTile<HabitFrequency>(
                        title: Text(freq.displayName),
                        value: freq,
                        contentPadding: EdgeInsets.zero,
                      ))
                  .toList(),
            ),
          ),
        ],
      ),
    );
  }

  static Color _hexToColor(String hex) {
    final value = int.parse(hex.replaceFirst('#', ''), radix: 16);
    return Color(value | 0xFF000000);
  }
}
