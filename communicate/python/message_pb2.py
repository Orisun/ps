# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: message.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf.internal import enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='message.proto',
  package='communicate',
  syntax='proto3',
  serialized_options=None,
  serialized_pb=_b('\n\rmessage.proto\x12\x0b\x63ommunicate\"S\n\x04Node\x12\r\n\x05Index\x18\x01 \x01(\x05\x12\x0c\n\x04Host\x18\x02 \x01(\t\x12\r\n\x05Ready\x18\x03 \x01(\x08\x12\x1f\n\x04Role\x18\x04 \x01(\x0e\x32\x11.communicate.Role\"#\n\x05Range\x12\r\n\x05\x42\x65gin\x18\x01 \x01(\x05\x12\x0b\n\x03\x45nd\x18\x02 \x01(\x05\")\n\x05Nodes\x12 \n\x05Nodes\x18\x01 \x03(\x0b\x32\x11.communicate.Node\"\x88\x01\n\x07\x43luster\x12\"\n\x07Servers\x18\x01 \x03(\x0b\x32\x11.communicate.Node\x12\x10\n\x08RangeEnd\x18\x02 \x03(\x05\x12#\n\x07Masters\x18\x03 \x03(\x0b\x32\x12.communicate.Nodes\x12\"\n\x06Slaves\x18\x04 \x03(\x0b\x32\x12.communicate.Nodes\"\x17\n\x05Value\x12\x0e\n\x06Values\x18\x01 \x03(\x01\"\xd4\x02\n\x07Message\x12\n\n\x02Id\x18\x01 \x01(\x03\x12!\n\x06Sender\x18\x02 \x01(\x0b\x32\x11.communicate.Node\x12#\n\x08Receiver\x18\x03 \x01(\x0b\x32\x11.communicate.Node\x12/\n\x11ServerClusterInfo\x18\x04 \x01(\x0b\x32\x14.communicate.Cluster\x12$\n\x08KeyRange\x18\x05 \x01(\x0b\x32\x12.communicate.Range\x12\"\n\x06Values\x18\x06 \x03(\x0b\x32\x12.communicate.Value\x12%\n\x07\x43ommand\x18\x07 \x01(\x0e\x32\x14.communicate.Command\x12\x14\n\x0cRespondMsgId\x18\x08 \x01(\x03\x12\x16\n\x0eRespondSuccess\x18\t \x01(\x08\x12%\n\nDeleteNode\x18\n \x01(\x0b\x32\x11.communicate.Node*D\n\x04Role\x12\x10\n\x0cUNKNOWN_ROLE\x10\x00\x12\x12\n\x0eSERVER_MANAGER\x10\x01\x12\n\n\x06SERVER\x10\x02\x12\n\n\x06WORKER\x10\x03*\xc5\x03\n\x07\x43ommand\x12\x13\n\x0fUNKNOWN_COMMAND\x10\x00\x12\x0e\n\nADD_SERVER\x10\x01\x12\x12\n\x0e\x41\x44\x44_SERVER_ACK\x10\x02\x12\x0f\n\x0bINIT_SERVER\x10\x03\x12\x13\n\x0fINIT_SERVER_ACK\x10\x04\x12\x11\n\rDELETE_SERVER\x10\x05\x12\x15\n\x11\x44\x45LETE_SERVER_ACK\x10\x06\x12\x14\n\x10KEY_RANGE_CHANGE\x10\x07\x12\x18\n\x14KEY_RANGE_CHANGE_ACK\x10\x08\x12\x17\n\x13MASTER_SLAVE_CHANGE\x10\t\x12\x1b\n\x17MASTER_SLAVE_CHANGE_ACK\x10\n\x12\x08\n\x04PULL\x10\x0b\x12\x0c\n\x08PULL_ACK\x10\x0c\x12\x08\n\x04PUSH\x10\r\x12\x0c\n\x08PUSH_ACK\x10\x0e\x12\n\n\x06UPDATE\x10\x0f\x12\x0e\n\nUPDATE_ACK\x10\x10\x12\x07\n\x03INC\x10\x11\x12\x0b\n\x07INC_ACK\x10\x12\x12\x07\n\x03\x41\x44\x44\x10\x13\x12\x0b\n\x07\x41\x44\x44_ACK\x10\x14\x12\x08\n\x04PING\x10\x15\x12\x0c\n\x08PING_ACK\x10\x16\x12\x18\n\x14\x43HANGE_SERVER_FINISH\x10\x17\x12\x0e\n\nADD_WORKER\x10\x18\x12\x11\n\rSERVER_CHANGE\x10\x19\x62\x06proto3')
)

_ROLE = _descriptor.EnumDescriptor(
  name='Role',
  full_name='communicate.Role',
  filename=None,
  file=DESCRIPTOR,
  values=[
    _descriptor.EnumValueDescriptor(
      name='UNKNOWN_ROLE', index=0, number=0,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='SERVER_MANAGER', index=1, number=1,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='SERVER', index=2, number=2,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='WORKER', index=3, number=3,
      serialized_options=None,
      type=None),
  ],
  containing_type=None,
  serialized_options=None,
  serialized_start=702,
  serialized_end=770,
)
_sym_db.RegisterEnumDescriptor(_ROLE)

Role = enum_type_wrapper.EnumTypeWrapper(_ROLE)
_COMMAND = _descriptor.EnumDescriptor(
  name='Command',
  full_name='communicate.Command',
  filename=None,
  file=DESCRIPTOR,
  values=[
    _descriptor.EnumValueDescriptor(
      name='UNKNOWN_COMMAND', index=0, number=0,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='ADD_SERVER', index=1, number=1,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='ADD_SERVER_ACK', index=2, number=2,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='INIT_SERVER', index=3, number=3,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='INIT_SERVER_ACK', index=4, number=4,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='DELETE_SERVER', index=5, number=5,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='DELETE_SERVER_ACK', index=6, number=6,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='KEY_RANGE_CHANGE', index=7, number=7,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='KEY_RANGE_CHANGE_ACK', index=8, number=8,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='MASTER_SLAVE_CHANGE', index=9, number=9,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='MASTER_SLAVE_CHANGE_ACK', index=10, number=10,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PULL', index=11, number=11,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PULL_ACK', index=12, number=12,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PUSH', index=13, number=13,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PUSH_ACK', index=14, number=14,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='UPDATE', index=15, number=15,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='UPDATE_ACK', index=16, number=16,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='INC', index=17, number=17,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='INC_ACK', index=18, number=18,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='ADD', index=19, number=19,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='ADD_ACK', index=20, number=20,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PING', index=21, number=21,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='PING_ACK', index=22, number=22,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='CHANGE_SERVER_FINISH', index=23, number=23,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='ADD_WORKER', index=24, number=24,
      serialized_options=None,
      type=None),
    _descriptor.EnumValueDescriptor(
      name='SERVER_CHANGE', index=25, number=25,
      serialized_options=None,
      type=None),
  ],
  containing_type=None,
  serialized_options=None,
  serialized_start=773,
  serialized_end=1226,
)
_sym_db.RegisterEnumDescriptor(_COMMAND)

Command = enum_type_wrapper.EnumTypeWrapper(_COMMAND)
UNKNOWN_ROLE = 0
SERVER_MANAGER = 1
SERVER = 2
WORKER = 3
UNKNOWN_COMMAND = 0
ADD_SERVER = 1
ADD_SERVER_ACK = 2
INIT_SERVER = 3
INIT_SERVER_ACK = 4
DELETE_SERVER = 5
DELETE_SERVER_ACK = 6
KEY_RANGE_CHANGE = 7
KEY_RANGE_CHANGE_ACK = 8
MASTER_SLAVE_CHANGE = 9
MASTER_SLAVE_CHANGE_ACK = 10
PULL = 11
PULL_ACK = 12
PUSH = 13
PUSH_ACK = 14
UPDATE = 15
UPDATE_ACK = 16
INC = 17
INC_ACK = 18
ADD = 19
ADD_ACK = 20
PING = 21
PING_ACK = 22
CHANGE_SERVER_FINISH = 23
ADD_WORKER = 24
SERVER_CHANGE = 25



_NODE = _descriptor.Descriptor(
  name='Node',
  full_name='communicate.Node',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Index', full_name='communicate.Node.Index', index=0,
      number=1, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Host', full_name='communicate.Node.Host', index=1,
      number=2, type=9, cpp_type=9, label=1,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Ready', full_name='communicate.Node.Ready', index=2,
      number=3, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Role', full_name='communicate.Node.Role', index=3,
      number=4, type=14, cpp_type=8, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=30,
  serialized_end=113,
)


_RANGE = _descriptor.Descriptor(
  name='Range',
  full_name='communicate.Range',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Begin', full_name='communicate.Range.Begin', index=0,
      number=1, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='End', full_name='communicate.Range.End', index=1,
      number=2, type=5, cpp_type=1, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=115,
  serialized_end=150,
)


_NODES = _descriptor.Descriptor(
  name='Nodes',
  full_name='communicate.Nodes',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Nodes', full_name='communicate.Nodes.Nodes', index=0,
      number=1, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=152,
  serialized_end=193,
)


_CLUSTER = _descriptor.Descriptor(
  name='Cluster',
  full_name='communicate.Cluster',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Servers', full_name='communicate.Cluster.Servers', index=0,
      number=1, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='RangeEnd', full_name='communicate.Cluster.RangeEnd', index=1,
      number=2, type=5, cpp_type=1, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Masters', full_name='communicate.Cluster.Masters', index=2,
      number=3, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Slaves', full_name='communicate.Cluster.Slaves', index=3,
      number=4, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=196,
  serialized_end=332,
)


_VALUE = _descriptor.Descriptor(
  name='Value',
  full_name='communicate.Value',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Values', full_name='communicate.Value.Values', index=0,
      number=1, type=1, cpp_type=5, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=334,
  serialized_end=357,
)


_MESSAGE = _descriptor.Descriptor(
  name='Message',
  full_name='communicate.Message',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='Id', full_name='communicate.Message.Id', index=0,
      number=1, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Sender', full_name='communicate.Message.Sender', index=1,
      number=2, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Receiver', full_name='communicate.Message.Receiver', index=2,
      number=3, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='ServerClusterInfo', full_name='communicate.Message.ServerClusterInfo', index=3,
      number=4, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='KeyRange', full_name='communicate.Message.KeyRange', index=4,
      number=5, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Values', full_name='communicate.Message.Values', index=5,
      number=6, type=11, cpp_type=10, label=3,
      has_default_value=False, default_value=[],
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='Command', full_name='communicate.Message.Command', index=6,
      number=7, type=14, cpp_type=8, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='RespondMsgId', full_name='communicate.Message.RespondMsgId', index=7,
      number=8, type=3, cpp_type=2, label=1,
      has_default_value=False, default_value=0,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='RespondSuccess', full_name='communicate.Message.RespondSuccess', index=8,
      number=9, type=8, cpp_type=7, label=1,
      has_default_value=False, default_value=False,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
    _descriptor.FieldDescriptor(
      name='DeleteNode', full_name='communicate.Message.DeleteNode', index=9,
      number=10, type=11, cpp_type=10, label=1,
      has_default_value=False, default_value=None,
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      serialized_options=None, file=DESCRIPTOR),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  serialized_options=None,
  is_extendable=False,
  syntax='proto3',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=360,
  serialized_end=700,
)

_NODE.fields_by_name['Role'].enum_type = _ROLE
_NODES.fields_by_name['Nodes'].message_type = _NODE
_CLUSTER.fields_by_name['Servers'].message_type = _NODE
_CLUSTER.fields_by_name['Masters'].message_type = _NODES
_CLUSTER.fields_by_name['Slaves'].message_type = _NODES
_MESSAGE.fields_by_name['Sender'].message_type = _NODE
_MESSAGE.fields_by_name['Receiver'].message_type = _NODE
_MESSAGE.fields_by_name['ServerClusterInfo'].message_type = _CLUSTER
_MESSAGE.fields_by_name['KeyRange'].message_type = _RANGE
_MESSAGE.fields_by_name['Values'].message_type = _VALUE
_MESSAGE.fields_by_name['Command'].enum_type = _COMMAND
_MESSAGE.fields_by_name['DeleteNode'].message_type = _NODE
DESCRIPTOR.message_types_by_name['Node'] = _NODE
DESCRIPTOR.message_types_by_name['Range'] = _RANGE
DESCRIPTOR.message_types_by_name['Nodes'] = _NODES
DESCRIPTOR.message_types_by_name['Cluster'] = _CLUSTER
DESCRIPTOR.message_types_by_name['Value'] = _VALUE
DESCRIPTOR.message_types_by_name['Message'] = _MESSAGE
DESCRIPTOR.enum_types_by_name['Role'] = _ROLE
DESCRIPTOR.enum_types_by_name['Command'] = _COMMAND
_sym_db.RegisterFileDescriptor(DESCRIPTOR)

Node = _reflection.GeneratedProtocolMessageType('Node', (_message.Message,), dict(
  DESCRIPTOR = _NODE,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Node)
  ))
_sym_db.RegisterMessage(Node)

Range = _reflection.GeneratedProtocolMessageType('Range', (_message.Message,), dict(
  DESCRIPTOR = _RANGE,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Range)
  ))
_sym_db.RegisterMessage(Range)

Nodes = _reflection.GeneratedProtocolMessageType('Nodes', (_message.Message,), dict(
  DESCRIPTOR = _NODES,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Nodes)
  ))
_sym_db.RegisterMessage(Nodes)

Cluster = _reflection.GeneratedProtocolMessageType('Cluster', (_message.Message,), dict(
  DESCRIPTOR = _CLUSTER,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Cluster)
  ))
_sym_db.RegisterMessage(Cluster)

Value = _reflection.GeneratedProtocolMessageType('Value', (_message.Message,), dict(
  DESCRIPTOR = _VALUE,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Value)
  ))
_sym_db.RegisterMessage(Value)

Message = _reflection.GeneratedProtocolMessageType('Message', (_message.Message,), dict(
  DESCRIPTOR = _MESSAGE,
  __module__ = 'message_pb2'
  # @@protoc_insertion_point(class_scope:communicate.Message)
  ))
_sym_db.RegisterMessage(Message)


# @@protoc_insertion_point(module_scope)