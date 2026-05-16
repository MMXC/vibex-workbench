# SpecPilot CLI Reference

## Quick Reference

### DataCenter (L2)
```
dc list                          # List all data keys
dc get <key>                     # Get single value
dc set <key> <value>             # Set value
dc watch <key>                   # Subscribe to changes (5s timeout)
dc query "SELECT *"              # SQL-style query
dc apply-result                  # Apply adapter query result to DC
```

### EventCenter (L3)
```
ec subscribe <event> <subscriber> # Subscribe component to event
ec emit <event> <payload_json>    # Emit event
ec history                        # View event history
```

### Adapter (L4)
```
ad list                           # List adapters
ad switch <name>                  # Switch active adapter
ad query <sql>                    # Query via adapter
ad test <name>                    # Test connection
```

### SpecRegistry (L4)
```
spec list                         # List specs
spec get <name>                   # Get spec detail
spec binding <name>               # Check field bindings
spec check <name>                 # Validate spec
```

### MFRegistry (L1)
```
mf list                           # List registered components
mf register <name> <path>         # Register component
mf resolve-from-spec <spec>        # Resolve components from spec
```
