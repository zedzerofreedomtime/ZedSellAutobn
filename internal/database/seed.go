package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const seedSQL = `
INSERT INTO vehicle_categories (slug, title, description, image_url, sort_order) VALUES
('all', 'รถทั้งหมด', 'รวมรถมือสองคัดเกรดทุกประเภทจาก Zed Auto', 'https://images.unsplash.com/photo-1549924231-f129b911e442?auto=format&fit=crop&w=1200&q=80', 0),
('suv', 'SUV', 'รถอเนกประสงค์ นั่งสบาย พื้นที่เยอะ เหมาะกับครอบครัว', 'https://images.unsplash.com/photo-1519641471654-76ce0107ad1b?auto=format&fit=crop&w=1200&q=80', 1),
('sedan', 'Sedan', 'รถซีดานพรีเมียม ขับนุ่ม ประหยัด และดูเป็นมืออาชีพ', 'https://images.unsplash.com/photo-1555215695-3004980ad54e?auto=format&fit=crop&w=1200&q=80', 2),
('ev', 'EV', 'รถไฟฟ้าไมล์สวย ค่าใช้จ่ายต่ำ เทคโนโลยีทันสมัย', 'https://images.unsplash.com/photo-1560958089-b8a1929cea89?auto=format&fit=crop&w=1200&q=80', 3),
('pickup', 'Pickup', 'รถกระบะพร้อมใช้งาน งานหนัก เดินทางไกล หรือใช้ธุรกิจ', 'https://images.unsplash.com/photo-1533473359331-0135ef1b58bf?auto=format&fit=crop&w=1200&q=80', 4),
('luxury', 'Luxury', 'รถหรูคัดพิเศษ ประวัติชัด พร้อมบริการดูแลระดับพรีเมียม', 'https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80', 5)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO vehicles (slug, category_slug, name, year, price_thb, monthly_payment_thb, location, mileage_km, fuel_type, tag, tone, image_url, gallery, transmission, drive_train, engine, exterior_color, interior_color, seats, owner_summary, description, seller_name, is_featured) VALUES
('2022-bmw-320d-m-sport', 'sedan', 'BMW 320d M Sport', 2022, 1689000, 28900, 'กรุงเทพฯ', 34000, 'ดีเซล', 'Top pick', 'success', 'https://images.unsplash.com/photo-1555215695-3004980ad54e?auto=format&fit=crop&w=1200&q=80', '["https://images.unsplash.com/photo-1555215695-3004980ad54e?auto=format&fit=crop&w=1400&q=80","https://images.unsplash.com/photo-1734554250249-1b54d0c2e570?auto=format&fit=crop&w=900&q=80","https://images.unsplash.com/photo-1689264048432-4788f2b14a19?auto=format&fit=crop&w=900&q=80"]', 'อัตโนมัติ', 'RWD', '2.0L TwinPower Turbo', 'Alpine White', 'Dakota Black', 5, 'เจ้าของเดิม 1 คน', 'รถสภาพสวย ภายในสะอาด ประวัติเข้าศูนย์ครบ เหมาะกับผู้ที่ต้องการรถยุโรปขับสนุกแต่ยังใช้งานได้ทุกวัน', 'Zed Certified Bangkok', TRUE),
('2023-audi-q5-sportback-quattro', 'suv', 'Audi Q5 Sportback quattro', 2023, 2490000, 42900, 'กรุงเทพฯ', 18000, 'เบนซิน', 'Featured', 'success', 'https://images.unsplash.com/photo-1492144534655-ae79c964c9d7?auto=format&fit=crop&w=1200&q=82', '["https://images.unsplash.com/photo-1492144534655-ae79c964c9d7?auto=format&fit=crop&w=1400&q=82","https://images.unsplash.com/photo-1673393663627-1cbca927ef18?auto=format&fit=crop&w=900&q=80","https://images.unsplash.com/photo-1723361527079-d9e33406fa8d?auto=format&fit=crop&w=900&q=80"]', 'S tronic 7 speed', 'quattro AWD', '2.0L TFSI', 'Glacier White', 'Leather Black', 5, 'เจ้าของเดิม 1 คน', 'รถไมล์น้อย ออปชันครบ พร้อมระบบขับเคลื่อน quattro และภาพลักษณ์สปอร์ตทันสมัย', 'Zed Auto Rama 9', TRUE),
('2022-tesla-model-3-long-range', 'ev', 'Tesla Model 3 Long Range', 2022, 1899000, 31600, 'ปทุมธานี', 29400, 'ไฟฟ้า', 'EV Deal', 'success', 'https://images.unsplash.com/photo-1560958089-b8a1929cea89?auto=format&fit=crop&w=1200&q=80', '["https://images.unsplash.com/photo-1560958089-b8a1929cea89?auto=format&fit=crop&w=1400&q=80","https://images.unsplash.com/photo-1694889649741-6054088247fc?auto=format&fit=crop&w=900&q=80","https://images.unsplash.com/photo-1694889650440-0e58c2db14c4?auto=format&fit=crop&w=900&q=80"]', 'Single speed', 'AWD', 'Dual motor electric', 'Pearl White', 'Premium White', 5, 'เจ้าของเดิม 1 คน', 'EV ระยะทางไกล แบตเตอรี่ดี ซอฟต์แวร์อัปเดตล่าสุด พร้อมใช้งานทันที', 'Zed EV Center', TRUE),
('2023-toyota-hilux-revo-rocco', 'pickup', 'Toyota Hilux Revo Rocco', 2023, 879000, 15200, 'ขอนแก่น', 21700, 'ดีเซล', 'Ready', 'warning', 'https://images.unsplash.com/photo-1533473359331-0135ef1b58bf?auto=format&fit=crop&w=1200&q=80', '["https://images.unsplash.com/photo-1533473359331-0135ef1b58bf?auto=format&fit=crop&w=1400&q=80","https://images.unsplash.com/photo-1675124516944-c257f7354c22?auto=format&fit=crop&w=900&q=80"]', 'อัตโนมัติ 6 speed', '4WD', '2.8L Diesel Turbo', 'Graphite', 'Black', 5, 'เจ้าของเดิม 1 คน', 'กระบะตัวท็อปพร้อมใช้งาน ช่วงล่างดี เหมาะทั้งใช้งานธุรกิจและเดินทางต่างจังหวัด', 'Zed Truck Khon Kaen', FALSE),
('2023-lexus-rx-350h-luxury', 'luxury', 'Lexus RX 350h Luxury', 2023, 3790000, 62400, 'กรุงเทพฯ', 13500, 'ไฮบริด', 'Premium', 'success', 'https://images.unsplash.com/photo-1606016159991-dfe4f2746ad5?auto=format&fit=crop&w=1200&q=80', '["https://images.unsplash.com/photo-1606016159991-dfe4f2746ad5?auto=format&fit=crop&w=1400&q=80","https://images.unsplash.com/photo-1741089040480-238da1bf915c?auto=format&fit=crop&w=900&q=80"]', 'E-CVT', 'AWD', '2.5L Hybrid', 'Sonic Titanium', 'Semi-aniline Brown', 5, 'เจ้าของเดิม 1 คน', 'SUV หรูขับสบาย เงียบ ประหยัด และดูแลง่าย เหมาะกับผู้บริหารและครอบครัว', 'Zed Luxury Bangkok', TRUE)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO pricing_highlights (label, value, sort_order) VALUES
('ค่าค้นหารถ', 'ฟรี', 1),
('ค่าจองเริ่มต้น', '฿5,000', 2),
('Success fee สำหรับผู้ขาย', 'เริ่ม 1.5%', 3)
ON CONFLICT DO NOTHING;

INSERT INTO pricing_plans (title, description, price_label, highlight, features, sort_order) VALUES
('Buyer Essential', 'สำหรับผู้ซื้อที่ต้องการค้นหา เปรียบเทียบ และคุยกับผู้ขายได้ทันที', '฿0', '', '["เข้าถึงรายการรถทั้งหมด","ดูรายงานและเปรียบเทียบราคาเบื้องต้น","คำนวณไฟแนนซ์และค่างวด","นัดดูรถผ่านระบบ"]', 1),
('Seller Assist', 'เหมาะกับเจ้าของรถที่ต้องการทีมช่วยปิดดีลและคัดกรองผู้ซื้อ', '1.5% เมื่อขายสำเร็จ', 'Popular', '["ไม่มีค่าลงประกาศล่วงหน้า","ทีมช่วยจัดหน้า listing ให้น่าเชื่อถือ","คัดกรองผู้สนใจและประสานการนัดหมาย","ช่วยดูขั้นตอนเอกสารและโอน"]', 2),
('Concierge Premium', 'แพ็กเกจครบสำหรับรถพรีเมียมที่ต้องการภาพลักษณ์และบริการแบบเต็มชุด', 'เริ่ม ฿12,900', '', '["ถ่ายภาพและจัดวางรายละเอียดแบบพรีเมียม","ตรวจสภาพเชิงลึกก่อนขาย","รับฝากขายและดูแลผู้ซื้อใกล้ชิด","มีตัวเลือกขนส่งถึงปลายทาง"]', 3)
ON CONFLICT (title) DO NOTHING;

INSERT INTO pricing_faqs (question, answer, sort_order) VALUES
('ค่าจองสามารถคืนได้หรือไม่?', 'โดยทั่วไปค่าจองขึ้นกับเงื่อนไขของดีลและสถานะการตรวจสภาพ เราแสดงเงื่อนไขให้ชัดเจนก่อนยืนยันทุกครั้ง', 1),
('ผู้ขายต้องจ่ายค่าลงประกาศหรือไม่?', 'แพ็กเกจมาตรฐานไม่มีค่าลงประกาศล่วงหน้า โดยคิดค่าบริการเมื่อปิดการขายสำเร็จ', 2),
('มีบริการช่วยเรื่องไฟแนนซ์หรือไม่?', 'มี ทีมงานช่วยประเมินวงเงินเบื้องต้นและแนะนำแผนผ่อนที่เหมาะกับลูกค้า', 3)
ON CONFLICT (question) DO NOTHING;

INSERT INTO how_it_works_steps (label, title, description, sort_order) VALUES
('Step 1', 'เลือกคันที่ใช่จากรถที่ผ่านการคัดเกรด', 'เริ่มจากการค้นหารถตามประเภท งบประมาณ หรือไลฟ์สไตล์ พร้อมดูราคากับไมล์ในหน้าเดียว', 1),
('Step 2', 'ตรวจรายละเอียด เปรียบเทียบ และคำนวณไฟแนนซ์', 'ดูภาพจริง สเปก รายงานรถ และประเมินค่างวดก่อนตัดสินใจ', 2),
('Step 3', 'จองดูรถหรือทดลองขับได้ทันที', 'เลือกเวลาที่สะดวก ส่งข้อเสนอ หรือเริ่มคุยกับผู้ขายผ่านระบบกลางของเรา', 3),
('Step 4', 'ปิดดีล ส่งเอกสาร และรับรถอย่างมั่นใจ', 'ทีมช่วยดูเรื่องเอกสาร การชำระเงิน และการส่งมอบรถให้ขั้นตอนสุดท้ายลื่นไหล', 4)
ON CONFLICT DO NOTHING;

INSERT INTO trust_signals (title, description, icon, sort_order) VALUES
('ข้อมูลรถชัดเจน', 'ดูประวัติรถ ภาพจริง และข้อมูลสำคัญก่อนตัดสินใจ', 'shield-check', 1),
('คุมงบได้ง่าย', 'มีเครื่องมือช่วยประเมินค่างวดและค่าใช้จ่ายรวม', 'wallet', 2),
('ปิดดีลเป็นขั้นตอน', 'ช่วยดูเรื่องนัดหมาย เอกสาร และการส่งมอบรถ', 'circle-check', 3)
ON CONFLICT DO NOTHING;

INSERT INTO experience_items (audience, content, sort_order) VALUES
('buyer', 'ค้นหารถจากสต็อกที่มีข้อมูลชัดเจนและภาพจริงหลายมุม', 1),
('buyer', 'เปรียบเทียบราคารถรุ่นใกล้เคียงก่อนตัดสินใจ', 2),
('buyer', 'เลือกแผนไฟแนนซ์เบื้องต้นและดูงบประมาณรายเดือน', 3),
('buyer', 'นัดดูรถ ทดลองขับ และสรุปดีลจากหน้ารายละเอียดรถได้เลย', 4),
('seller', 'ลงข้อมูลรถพร้อมทีมช่วยจัดภาพและรายละเอียดให้น่าเชื่อถือ', 1),
('seller', 'คัดกรองผู้สนใจก่อนนัดหมายเพื่อลดเวลาที่เสียไป', 2),
('seller', 'มีทีมช่วยประสานเรื่องเอกสารและการชำระเงิน', 3),
('seller', 'เพิ่มบริการตรวจสภาพและส่งมอบรถได้ตามต้องการ', 4)
ON CONFLICT DO NOTHING;

INSERT INTO blog_posts (slug, category, title, excerpt, image_url, published_at, read_time_minutes, author, sections, is_featured) VALUES
('how-to-budget-for-your-next-used-car', 'Finance', 'วางงบซื้อรถมือสองอย่างไรไม่ให้ผ่อนตึงเกินไป', 'เริ่มจากค่างวดที่รับได้จริง แล้วค่อยย้อนกลับมาหาช่วงราคารถที่เหมาะสม', 'https://images.unsplash.com/photo-1554224155-6726b3ff858f?auto=format&fit=crop&w=1400&q=80', '2026-04-18', 5, 'Zed Editorial', '[{"heading":"เริ่มจากค่างวดที่ไหว","body":["หลายคนเริ่มจากวงเงินกู้ แต่จุดที่ควรเริ่มจริงคือค่างวดที่รับได้โดยไม่กดดันงบส่วนอื่น","เมื่อรู้ค่างวดที่สบายแล้ว ค่อยย้อนกลับมาหาช่วงราคารถที่เหมาะจะช่วยให้ตัดสินใจง่ายขึ้น"]},{"heading":"เตรียมเอกสารให้พร้อมก่อนคุยไฟแนนซ์","body":["เอกสารอย่างบัตรประชาชน สลิปเงินเดือน และรายการเดินบัญชี มักเป็นจุดที่ทำให้ขั้นตอนช้าหากเตรียมไม่ครบ","การเตรียมล่วงหน้าช่วยให้รู้ผลประเมินเร็วขึ้น และต่อรองดีลได้มั่นใจกว่าเดิม"]}]', TRUE),
('ev-battery-health-questions-to-ask-before-buying', 'EV', 'ซื้อ EV มือสองต้องถามอะไรเรื่องแบตเตอรี่บ้าง', 'รวมคำถามสำคัญเกี่ยวกับ battery health ประวัติการชาร์จ และการรับประกันที่ควรเช็กก่อนปิดดีล', 'https://images.unsplash.com/photo-1593941707882-a5bba53b3f87?auto=format&fit=crop&w=1400&q=80', '2026-04-12', 7, 'Zed EV Center', '[{"heading":"ถามค่า battery health ให้ชัด","body":["ค่า battery health เป็นตัวชี้วัดสำคัญว่ารถยังเก็บพลังงานได้ดีแค่ไหน","รถ EV มือสองที่ดูใหม่มากอาจมีสภาพแบตต่างกันได้เยอะ จึงควรขอหลักฐานประกอบทุกครั้ง"]},{"heading":"เช็กประกันที่ยังเหลือ","body":["รถ EV หลายรุ่นยังเหลือประกันแบตจากผู้ผลิต ซึ่งเป็นข้อได้เปรียบมากเวลาเลือกรถมือสอง","ควรตรวจทั้งระยะเวลา เงื่อนไขเคลม และผลของการเปลี่ยนเจ้าของ"]}]', FALSE),
('sedan-vs-suv-which-one-fits-city-life-better', 'Comparison', 'Sedan หรือ SUV แบบไหนเหมาะกับชีวิตในเมืองมากกว่า', 'เปรียบเทียบความคล่องตัว พื้นที่ใช้สอย และต้นทุนเพื่อช่วยเลือกให้เหมาะกับการใช้งานจริง', 'https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1400&q=80', '2026-04-05', 4, 'Zed Editorial', '[{"heading":"Sedan เด่นเรื่องความคล่อง","body":["Sedan ให้ฟีลขับนิ่งและคล่องกว่าในเมือง โดยเฉพาะเวลาเข้าซอยหรือจอด","ถ้าใช้งานคนเดียวหรือเดินทางเป็นคู่ มักเป็นตัวเลือกที่คุ้มและดูแลง่าย"]},{"heading":"SUV เด่นเรื่องพื้นที่และมุมมอง","body":["SUV เหมาะกับคนที่มีครอบครัวหรือขนของบ่อย เพราะพื้นที่ใช้สอยและท่านั่งตอบโจทย์กว่า","ถ้าชอบท่านั่งสูงและเข้าออกง่าย SUV มักให้ประสบการณ์สบายกว่า"]}]', FALSE)
ON CONFLICT (slug) DO NOTHING;
`

func Seed(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, seedSQL)
	return err
}
