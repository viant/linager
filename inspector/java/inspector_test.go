package java_test

import (
	"testing"

	"github.com/viant/linager/inspector/info"
	"github.com/viant/linager/inspector/java"
)

func TestInspector_InspectSource(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantName string
		wantErr  bool
	}{
		{
			name: "Simple class",
			source: `package com.example;
@SuppressWarnings("unused")
public class Person {
    private String name;
    private int age;
    
    public Person(String name, int age) {
        this.name = name;
        this.age = age;
    }
    
    public String getName() {
        return name;
    }
    
    public int getAge() {
        return age;
    }
}`,
			wantName: "com.example",
			wantErr:  false,
		},
		{
			name: "Interface",
			source: `package com.example.interfaces;

public interface UserService {
    User findById(Long id);
    List<User> findAll();
    User save(User user);
    void delete(Long id);
}`,
			wantName: "com.example.interfaces",
			wantErr:  false,
		},
		{
			name: "Enum",
			source: `package com.example.enums;

public enum Day {
    MONDAY, TUESDAY, WEDNESDAY, THURSDAY, FRIDAY, SATURDAY, SUNDAY
}`,
			wantName: "com.example.enums",
			wantErr:  false,
		},
		{
			name: "Complex class with generics",
			source: `package com.example.collections;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;

public class CustomList<E> implements Collection<E> {
    private ArrayList<E> internal = new ArrayList<>();
    
    @Override
    public int size() {
        return internal.size();
    }
    
    @Override
    public boolean add(E element) {
        return internal.add(element);
    }
    
    @Override
    public Iterator<E> iterator() {
        return internal.iterator();
    }
}`,
			wantName: "com.example.collections",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := java.NewInspector(&info.Config{IncludeUnexported: true})
			pkg, err := inspector.InspectSource([]byte(tt.source))
			if (err != nil) != tt.wantErr {
				t.Errorf("Inspector.InspectSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if pkg == nil {
				if !tt.wantErr {
					t.Errorf("Inspector.InspectSource() returned nil package, expected non-nil")
				}
				return
			}

			if pkg.Name != tt.wantName {
				t.Errorf("Inspector.InspectSource() package name = %s, want %s", pkg.Name, tt.wantName)
			}

			// Basic validation to ensure we got something back
			if len(pkg.Types) == 0 {
				t.Errorf("Inspector.InspectSource() returned zero types")
			}

			for _, typ := range pkg.Types {
				if typ.Name == "" {
					t.Errorf("Found a type with empty name")
				}
			}
		})
	}
}

func TestInspector_InspectFile(t *testing.T) {
	// This test requires actual Java files on disk, so we'll skip it
	t.Skip("Skipping file-based tests - requires Java files on disk")
}

func TestInspector_InspectPackage(t *testing.T) {
	// This test requires actual Java packages on disk, so we'll skip it
	t.Skip("Skipping package-based tests - requires Java packages on disk")
}
