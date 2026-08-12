import datetime
from uuid import UUID


def hexuid_to_decimal(uuid):
    if not isinstance(uuid, str) and not isinstance(uuid, UUID):
        uuid = str(uuid)
    if isinstance(uuid, str):
        hex_part = uuid.split("-")[0]
        decimal_number = int(hex_part, 16)
        return str(decimal_number)
    elif isinstance(uuid, UUID):
        return str(uuid.int)


def _deep_value(node):
    """从 {'value': x} 这类 GVAS 节点中递归取出纯值（Guid/str/int）。"""
    seen = 0
    while isinstance(node, dict) and "value" in node and seen < 6:
        node = node["value"]
        seen += 1
    return node


def _extract_platform_id(platform_node):
    """安全提取玩家 PlatformID（不同版本结构可能不同）。"""
    if not platform_node:
        return ""
    try:
        val = _deep_value(platform_node)
        if isinstance(val, dict):
            for key in ("platform_user_id", "ID", "id", "OnlinePlatform"):
                if key in val:
                    sub = _deep_value(val[key])
                    if sub:
                        return str(sub)
            return ""
        return str(val) if val is not None else ""
    except Exception:
        return ""


def tick2local(tick, real_date_time_ticks, filetime):
    ts = filetime + (tick - real_date_time_ticks) / 1e7
    # to RFC3339 like 2006-01-02T15:04:05Z07:00
    t = datetime.datetime.fromtimestamp(ts, tz=datetime.timezone.utc)
    return t.strftime("%Y-%m-%dT%H:%M:%SZ%z").replace("+0000", "")


class Player:
    def __init__(self, uid, data):
        self.player_uid = hexuid_to_decimal(uid)
        self.nickname = data["NickName"]["value"] if data.get("NickName") else ""
        # 平台 ID（Steam 为 steam_xxx，Xbox/GDK 为 gdk_xxx），存为原始字符串
        self.platform_id = _extract_platform_id(data.get("PlatformID"))
        self.level = int(data["Level"]["value"]) if data.get("Level") else 1
        self.exp = int(data["Exp"]["value"]) if data.get("Exp") else 0
        self.hp = int(data["HP"]["value"]["Value"]["value"]) if data.get("HP") else 0
        self.max_hp = (
            int(data["MaxHP"]["value"]["Value"]["value"]) if data.get("MaxHP") else 0
        )
        self.shield_hp = (
            int(data["ShieldHP"]["value"]["Value"]["value"])
            if data.get("ShieldHP")
            else 0
        )
        self.shield_max_hp = (
            int(data["ShieldMaxHP"]["value"]["Value"]["value"])
            if data.get("ShieldMaxHP")
            else 0
        )
        self.max_status_point = (
            int(data["MaxSP"]["value"]["Value"]["value"]) if data.get("MaxSP") else 0
        )
        self.status_point = {
            s["StatusName"]["value"]: s["StatusPoint"]["value"]
            for s in data["GotStatusPointList"]["value"]["values"]
        } if data.get("GotStatusPointList") else {}
        full_stomach = (
            float(data["FullStomach"]["value"]) if data.get("FullStomach") else 0
        )
        self.full_stomach = round(full_stomach, 2)
        self.pals = []
        self.items = (
            data["Items"]
            if data.get("Items") is not None
            else {
                "CommonContainerId": [],
                "DropSlotContainerId": [],
                "EssentialContainerId": [],
                "FoodEquipContainerId": [],
                "PlayerEquipArmorContainerId": [],
                "WeaponLoadOutContainerId": [],
            }
        )

        self.__order = [
            "player_uid",
            "nickname",
            "platform_id",
            "level",
            "exp",
            "hp",
            "max_hp",
            "shield_hp",
            "shield_max_hp",
            "max_status_point",
            "status_point",
            "full_stomach",
            "pals",
            "items",
        ]

    def to_dict(self):
        return {
            attr: getattr(self, attr)
            for attr in self.__order
            if not attr.startswith("_") and not callable(getattr(self, attr))
        }


def _get_int(data, key, default=0):
    """安全读取整数字段，兼容直接值与 {'value': {'Value': value}} 两种结构。"""
    node = data.get(key)
    if not node:
        return default
    if isinstance(node, dict) and "value" in node:
        val = node["value"]
        if isinstance(val, dict) and "Value" in val:
            return int(val["Value"]) if val["Value"] is not None else default
        if isinstance(val, (int, float)):
            return int(val)
        return default
    return int(node)


class Pal:
    def __init__(self, data, instance_id=None):
        self.owner = hexuid_to_decimal(data["OwnerPlayerUId"]["value"])
        # 存档内唯一实例 ID（用于复制检测）
        self.instance_id = str(instance_id) if instance_id else ""
        # self.nickname = data["Nickname"]["value"] if data.get("Nicknme") else ""
        self.level = int(data["Level"]["value"]) if data.get("Level") else 1
        self.exp = int(data["Exp"]["value"]) if data.get("Exp") else 0
        self.hp = int(data["HP"]["value"]["Value"]["value"]) if data.get("HP") else 0
        self.max_hp = (
            int(data["MaxHP"]["value"]["Value"]["value"]) if data.get("MaxHP") else 0
        )
        self.gender = (
            data["Gender"]["value"]["value"].split("::")[-1]
            if data.get("Gender")
            else "Unknow"
        )
        self.is_lucky = data["IsRarePal"]["value"] if data.get("IsRarePal") else False
        self.is_boss = False

        if data.get("CharacterID"):
            typename = data["CharacterID"]["value"]
            typename_upper = typename.upper()
            if typename_upper[:5] == "BOSS_":
                typename_upper = typename_upper.replace("BOSS_", "")
                self.is_boss = not self.is_lucky
            self.is_tower = typename_upper.startswith("GYM_")
            self.type = typename
        else:
            self.is_tower = False
            self.type = "Unknow"

        # 工作速度
        self.workspeed = data["CraftSpeed"]["value"] if data.get("CraftSpeed") else 0

        # 注意：Talent_* 是「个体值/天赋(IV)」，取值 0~100，并非实际战斗属性。
        # 参考项目误把它们当成 melee/ranged/defense，这里修正为 talent_* 字段。
        self.talent_hp = _get_int(data, "Talent_HP")
        self.talent_melee = _get_int(data, "Talent_Melee")
        self.talent_ranged = _get_int(data, "Talent_Shot")
        self.talent_defense = _get_int(data, "Talent_Defense")

        # 帕鲁灵魂强化(PalSouls)。不同版本字段名可能不同，做安全读取。
        self.soul_hp = _get_int(data, "Talent_MaxHP")
        self.soul_atk = _get_int(data, "Talent_Attack")
        self.soul_def = _get_int(data, "Talent_Defense")
        self.soul_cs = _get_int(data, "Talent_CraftSpeed")

        # 兼容字段：保留旧字段名，数值取实际战斗属性（存档中通常不直接保存，暂以 0 占位）
        self.melee = _get_int(data, "MeleeAttack")
        self.ranged = _get_int(data, "ShotAttack")
        self.defense = _get_int(data, "Defense")

        # 强化阶级（凝聚帕鲁次数）
        self.rank = int(data["Rank"]["value"]) if data.get("Rank") else 1
        # 被动技能词条（全量，正常最多 4 个）
        self.skills = (
            data["PassiveSkillList"]["value"]["values"]
            if data.get("PassiveSkillList")
            else []
        )
        # 已学会的主动技能（装备技能存在 LearnedAttacks/EquippedWaza）
        self.equipped_skills = []
        if data.get("EquippedWaza"):
            try:
                self.equipped_skills = list(data["EquippedWaza"]["value"].get("values", []))
            except Exception:
                self.equipped_skills = []

        self.__order = [
            "owner",
            "instance_id",
            "level",
            "exp",
            "hp",
            "max_hp",
            "type",
            "gender",
            "is_lucky",
            "is_boss",
            "is_tower",
            "workspeed",
            "talent_hp",
            "talent_melee",
            "talent_ranged",
            "talent_defense",
            "soul_hp",
            "soul_atk",
            "soul_def",
            "soul_cs",
            "melee",
            "ranged",
            "defense",
            "rank",
            "skills",
            "equipped_skills",
        ]

    def to_dict(self):
        return {
            attr: getattr(self, attr)
            for attr in self.__order
            if not attr.startswith("_") and not callable(getattr(self, attr))
        }


class Guild:
    def __init__(self, data, real_date_time_ticks, filetime):
        self.name = data["guild_name"]
        self.base_camp_level = data["base_camp_level"]
        self.admin_player_uid = hexuid_to_decimal(data["admin_player_uid"])
        self.players = [
            {
                "player_uid": hexuid_to_decimal(player["player_uid"]),
                "nickname": player["player_info"]["player_name"],
                "last_online": (
                    tick2local(
                        player["player_info"]["last_online_real_time"],
                        real_date_time_ticks,
                        filetime,
                    )
                    if player["player_info"].get("last_online_real_time")
                    else ""
                ),
            }
            for player in data["players"]
        ]
        self.base_ids = [str(x) for x in data["base_ids"]]
        self.__order = [
            "name",
            "base_camp_level",
            "admin_player_uid",
            "players",
            "base_ids",
        ]

    def to_dict(self):
        return {
            attr: getattr(self, attr)
            for attr in self.__order
            if not attr.startswith("_") and not callable(getattr(self, attr))
        }
